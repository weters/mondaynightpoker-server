package passthepoop

import (
	"encoding/json"
	"fmt"
	"strconv"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"
)

// persistedLog mirrors the JSON written by GameLog.
type persistedLog struct {
	Edition string           `json:"edition"`
	Pot     int              `json:"pot"`
	Ante    int              `json:"ante"`
	Rounds  []persistedRound `json:"rounds"`
	Players []int64          `json:"players"`
	Winner  int64            `json:"winner"`
}

type persistedRound struct {
	Round        int               `json:"round"`
	StartingHand []persistedHand   `json:"startingHand"`
	GameActions  []persistedAction `json:"gameActions"`
}

type persistedHand struct {
	PlayerID int64      `json:"playerId"`
	Card     *deck.Card `json:"card"`
}

type persistedAction struct {
	GameAction gamelog.RawID `json:"gameAction"`
	PlayerID   int64         `json:"playerId"`
}

// ParseGameLog decodes a persisted Pass the Poop log into a normalized hand.
//
// Pass the Poop has no betting: players ante once and then decide only whether to
// keep their card or trade it. There is nothing to wager and nothing to fold, so
// two of the normalized fields carry a weaker meaning here than they do for the
// poker variants, and that difference is deliberate rather than an omission:
//
//   - VoluntarilyPlayed is true when the player took a discretionary action —
//     staying, trading, blocking, or going to the deck. It measures engagement
//     rather than money committed by choice.
//   - Folded is always false. Losing a life is not a fold; a player cannot choose
//     to give up a round.
//
// A player reaches the showdown in every round they survive to, since every
// surviving card is compared, and the game's single Winner is the last player
// standing.
func ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error) {
	var log persistedLog
	if err := json.Unmarshal(raw, &log); err != nil {
		return nil, fmt.Errorf("could not decode pass the poop log: %w", err)
	}

	hand := &gamelog.Hand{
		Variant:   log.Edition,
		AnteCents: log.Ante,
		PotCents:  log.Pot,
		Rounds:    len(log.Rounds),
	}

	for _, id := range log.Players {
		hand.Participant(id)
	}

	acted := make(map[int64]bool)

	for _, round := range log.Rounds {
		street := "round-" + strconv.Itoa(round.Round)

		if round.Round == 0 {
			for _, h := range round.StartingHand {
				hand.Participant(h.PlayerID).StartingCards = []*deck.Card{h.Card}
			}
		}

		for _, a := range round.GameActions {
			kind, voluntary := actionKind(a.GameAction)
			if voluntary {
				acted[a.PlayerID] = true
			}

			hand.AddAction(&gamelog.Action{
				Street:   street,
				PlayerID: a.PlayerID,
				Kind:     kind,
			})
		}

		// Everyone dealt into a round contested it: cards are compared, not bet on.
		for _, h := range round.StartingHand {
			hand.Participant(h.PlayerID).WentToShowdown = true
		}
	}

	for _, p := range hand.Participants {
		p.VoluntarilyPlayed = acted[p.PlayerID]
	}

	if log.Winner != 0 {
		hand.Participant(log.Winner).Won = true
	}

	return hand, nil
}

// actionKind maps a Pass the Poop action identifier onto a normalized kind and
// reports whether it was a discretionary choice. Accepting a trade and drawing
// from the deck are forced consequences of another player's move, so they do not
// count as discretionary.
//
// The identifier is resolved through GameActionFromID and matched against the
// typed constants rather than against string literals, so renaming an action in
// GameAction.ID is a compile error here instead of silently reclassifying every
// action in every persisted log.
func actionKind(raw gamelog.RawID) (kind gamelog.Kind, voluntary bool) {
	a, err := GameActionFromID(string(raw))
	if err != nil {
		return gamelog.KindPass, false
	}

	switch a {
	case ActionStay:
		return gamelog.KindStayIn, true
	case ActionTrade, ActionBlockTrade, ActionFlipKing, ActionGoToDeck:
		return gamelog.KindTrade, true
	case ActionAccept, ActionDrawFromDeck:
		return gamelog.KindTrade, false
	}

	return gamelog.KindPass, false
}
