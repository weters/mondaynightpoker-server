package sevencard

import (
	"encoding/json"
	"fmt"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"
)

// persistedLog mirrors the JSON written by gameLog. See the equivalent type in the
// texasholdem package for why the persisted form is declared separately from the
// live struct.
type persistedLog struct {
	Variant    string                  `json:"variant"`
	Ante       int                     `json:"ante"`
	Seats      []persistedSeat         `json:"seats"`
	Deals      []persistedDeal         `json:"deals"`
	Actions    []gamelog.BettingAction `json:"actions"`
	FinalState persistedState          `json:"finalState"`
}

type persistedSeat struct {
	PlayerID int64 `json:"playerId"`
}

type persistedDealCard struct {
	PlayerID int64      `json:"playerId"`
	Card     *deck.Card `json:"card"`
}

type persistedDeal struct {
	Cards []persistedDealCard `json:"cards"`
}

type persistedState struct {
	Participants []persistedPart `json:"participants"`
	Pot          int             `json:"pot"`
	Winners      map[int64]int   `json:"winners"`
}

type persistedPart struct {
	PlayerID int64        `json:"playerId"`
	DidFold  bool         `json:"didFold"`
	Hand     []*deck.Card `json:"hand"`
}

// ParseGameLog decodes a persisted Seven Card log into a normalized hand.
//
// Seven Card has no single starting hand: cards arrive one per street, some face
// up and some face down. The starting cards reported for each player are therefore
// the cards from the initial deal only, reassembled from the deal records.
//
// The action stream carries more than the betting: a variant that implements
// InteractiveVariant records its own moves into the same list, in the order they
// were taken, so the parser hands ApplyBettingActionsFunc a mapping for them (see
// variantActionKind) rather than letting them fall out of the history.
func ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error) {
	var log persistedLog
	if err := json.Unmarshal(raw, &log); err != nil {
		return nil, fmt.Errorf("could not decode seven card log: %w", err)
	}

	hand := &gamelog.Hand{
		Variant:   log.Variant,
		AnteCents: log.Ante,
		PotCents:  log.FinalState.Pot,
		Rounds:    1,
	}

	for _, seat := range log.Seats {
		hand.Participant(seat.PlayerID)
	}

	if len(log.Deals) > 0 {
		for _, c := range log.Deals[0].Cards {
			p := hand.Participant(c.PlayerID)
			p.StartingCards = append(p.StartingCards, c.Card)
		}
	}

	hand.ApplyBettingActionsFunc(log.Actions, variantActionKind)

	folded := make(map[int64]bool, len(log.FinalState.Participants))
	for _, p := range log.FinalState.Participants {
		folded[p.PlayerID] = p.DidFold
		hand.Participant(p.PlayerID).FinalCards = p.Hand
	}
	hand.ResolveShowdown(folded, log.FinalState.Winners)

	return hand, nil
}

// variantActionKind maps a Seven Card variant action identifier onto a normalized
// kind. It is the fallback ParseGameLog gives ApplyBettingActionsFunc, so it only
// ever sees identifiers that are not part of the shared betting vocabulary.
//
// Chiggs is the one variant with actions of its own: a player may flip a face-down
// mushroom to force their neighbors out, and a neighbor holding an antidote may
// play it to survive. Both are recorded by Game.Action into the same stream as the
// betting and interleaved with it, so dropping them would leave a hand history that
// silently skips the moves that decided the hand.
//
// Both normalize to KindPlayCard. Each commits a specific card from the player's
// hand to change the state of the game, which is what KindPlayCard describes; the
// alternatives are worse fits, since neither trades or discards a card for another
// and neither is a wager. Neither puts money in the pot, and
// ApplyBettingActionsFunc guarantees that an action resolved here never counts
// toward VoluntarilyPlayed — the amount recorded for both is zero, so the wagered
// totals are unaffected as well.
//
// The switch matches the typed constants rather than string literals, so renaming
// one in action.go is a compile error here instead of an action that quietly
// disappears from every hand history written afterward.
func variantActionKind(raw gamelog.RawID) (gamelog.Kind, bool) {
	switch Action(raw) {
	case ActionFlipMushroom, ActionPlayAntidote:
		return gamelog.KindPlayCard, true
	}

	return "", false
}
