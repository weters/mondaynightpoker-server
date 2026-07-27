package texasholdem

import (
	"encoding/json"
	"fmt"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"
)

// persistedLog mirrors the JSON written by gameLog. It is declared separately
// rather than reusing gameLog because the persisted form is an external contract:
// re-decoding into the live struct would silently break the moment a field is
// renamed for the in-memory game, and because Variant and the action identifiers
// do not round-trip through their own MarshalJSON.
type persistedLog struct {
	Variant      gamelog.RawID           `json:"variant"`
	Ante         int                     `json:"ante"`
	Seats        []persistedSeat         `json:"seats"`
	Actions      []gamelog.BettingAction `json:"actions"`
	Community    []*deck.Card            `json:"community"`
	Pot          int                     `json:"pot"`
	Participants []persistedPart         `json:"participants"`
}

type persistedSeat struct {
	PlayerID  int64        `json:"playerId"`
	HoleCards []*deck.Card `json:"holeCards"`
}

type persistedPart struct {
	PlayerID int64        `json:"playerId"`
	Cards    []*deck.Card `json:"cards"`
	Folded   bool         `json:"folded"`
	Winnings int          `json:"winnings"`
}

// ParseGameLog decodes a persisted Texas Hold'em log into a normalized hand.
func ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error) {
	var log persistedLog
	if err := json.Unmarshal(raw, &log); err != nil {
		return nil, fmt.Errorf("could not decode texas hold'em log: %w", err)
	}

	hand := &gamelog.Hand{
		Variant:   string(log.Variant),
		AnteCents: log.Ante,
		PotCents:  log.Pot,
		Rounds:    1,
		Board:     log.Community,
	}

	for _, seat := range log.Seats {
		hand.Participant(seat.PlayerID).StartingCards = seat.HoleCards
	}

	hand.ApplyBettingActions(log.Actions)

	folded := make(map[int64]bool, len(log.Participants))
	winnings := make(map[int64]int, len(log.Participants))
	for _, p := range log.Participants {
		folded[p.PlayerID] = p.Folded
		winnings[p.PlayerID] = p.Winnings
		hand.Participant(p.PlayerID).FinalCards = p.Cards
	}
	hand.ResolveShowdown(folded, winnings)

	return hand, nil
}
