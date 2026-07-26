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

	hand.ApplyBettingActions(log.Actions)

	folded := make(map[int64]bool, len(log.FinalState.Participants))
	for _, p := range log.FinalState.Participants {
		folded[p.PlayerID] = p.DidFold
		hand.Participant(p.PlayerID).FinalCards = p.Hand
	}
	hand.ResolveShowdown(folded, log.FinalState.Winners)

	return hand, nil
}
