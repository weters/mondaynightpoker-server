package bourre

import (
	"encoding/json"
	"fmt"
	"strconv"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"
)

// persistedLog mirrors the JSON written by gameLog. Parent links to the previous
// hand when a Bourré game continues across several hands, so a single persisted
// log can carry a whole chain.
type persistedLog struct {
	Parent    *persistedLog      `json:"parent"`
	Ante      int                `json:"ante"`
	TrumpCard *deck.Card         `json:"trumpCard"`
	Seats     []persistedSeat    `json:"seats"`
	Discards  []persistedDiscard `json:"discards"`
	Tricks    []persistedTrick   `json:"tricks"`
	Result    *persistedResult   `json:"result"`
}

type persistedSeat struct {
	PlayerID     int64        `json:"playerId"`
	StartingHand []*deck.Card `json:"startingHand"`
}

type persistedDiscard struct {
	PlayerID  int64        `json:"playerId"`
	Folded    bool         `json:"folded"`
	Discarded []*deck.Card `json:"discarded"`
}

type persistedTrick struct {
	Number int             `json:"number"`
	Plays  []persistedPlay `json:"plays"`
}

type persistedPlay struct {
	PlayerID int64      `json:"playerId"`
	Card     *deck.Card `json:"card"`
}

// persistedResult mirrors bourre.Result. Its player slices hold the Player struct,
// which has no JSON tags and exports only PlayerID, so it serializes with the Go
// field name.
type persistedResult struct {
	Winners []persistedPlayer `json:"Winners"`
	OldPot  int               `json:"OldPot"`
}

type persistedPlayer struct {
	PlayerID int64 `json:"PlayerID"`
}

// ParseGameLog decodes a persisted Bourré log into a normalized hand.
//
// Bourré's fold is a decision made at the trade-in phase, before any trick is
// played, so it maps to a drop-out rather than a poker fold: the player forfeits
// their claim on the pot but nothing they have already wagered. Everyone who did
// not fold plays out every trick, so a non-folding player always reaches the
// showdown when at least one opponent also stayed in.
//
// Only the final hand of a continuation chain is normalized. Parent hands are
// counted toward Rounds so the chain length is visible, but their tricks are not
// merged in: each hand in the chain was its own deal with its own pot, and
// flattening them would double-count every player's participation.
func ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error) {
	var log persistedLog
	if err := json.Unmarshal(raw, &log); err != nil {
		return nil, fmt.Errorf("could not decode bourre log: %w", err)
	}

	rounds := 1
	for parent := log.Parent; parent != nil; parent = parent.Parent {
		rounds++
	}

	hand := &gamelog.Hand{
		AnteCents: log.Ante,
		Rounds:    rounds,
	}

	if log.TrumpCard != nil {
		hand.Board = []*deck.Card{log.TrumpCard}
	}
	if log.Result != nil {
		hand.PotCents = log.Result.OldPot
	}

	for _, seat := range log.Seats {
		hand.Participant(seat.PlayerID).StartingCards = seat.StartingHand
	}

	folded := make(map[int64]bool, len(log.Discards))
	for _, d := range log.Discards {
		if d.Folded {
			// AddAction records the fold; the map is what ResolveShowdown needs.
			folded[d.PlayerID] = true
			hand.AddAction(&gamelog.Action{
				Street:   "trade-in",
				PlayerID: d.PlayerID,
				Kind:     gamelog.KindDropOut,
			})
			continue
		}

		hand.AddAction(&gamelog.Action{
			Street:   "trade-in",
			PlayerID: d.PlayerID,
			Kind:     gamelog.KindDiscard,
			Cards:    d.Discarded,
		})
	}

	for _, trick := range log.Tricks {
		street := "trick-" + strconv.Itoa(trick.Number)
		for _, play := range trick.Plays {
			hand.AddAction(&gamelog.Action{
				Street:   street,
				PlayerID: play.PlayerID,
				Kind:     gamelog.KindPlayCard,
				Cards:    []*deck.Card{play.Card},
			})
		}
	}

	// Folding is the only way out of a Bourré hand, so the shared showdown rule
	// applies unchanged: everyone who did not fold contested it.
	//
	// The winnings map is nil because Bourré names its winners directly, and that
	// list is authoritative even when the amount is zero — a hand can be won for
	// nothing when the pot carries. Deriving Won from a payout would lose those.
	hand.ResolveShowdown(folded, nil)

	// Staying in past the trade-in is the commitment Bourré asks for; there is no
	// separate wager to opt into.
	for _, p := range hand.Participants {
		p.VoluntarilyPlayed = !p.Folded
	}

	if log.Result != nil {
		for _, w := range log.Result.Winners {
			hand.Participant(w.PlayerID).Won = true
		}
	}

	return hand, nil
}
