package littlel

import (
	"encoding/json"
	"fmt"
	"strconv"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"
)

// persistedLog mirrors the JSON written by gameLog. See the equivalent type in the
// texasholdem package for why the persisted form is declared separately from the
// live struct.
type persistedLog struct {
	Ante         int             `json:"ante"`
	Seats        []persistedSeat `json:"seats"`
	Community    []*deck.Card    `json:"community"`
	Actions      []persistedAct  `json:"actions"`
	Participants []persistedPart `json:"participants"`
	Winners      map[int64]int   `json:"winners"`
}

type persistedSeat struct {
	PlayerID     int64        `json:"playerId"`
	StartingHand []*deck.Card `json:"startingHand"`
}

// persistedAct embeds the shared betting action and adds the one field Little L
// records differently: a plain integer round rather than a named street. Street is
// left empty by the decode and filled in below.
type persistedAct struct {
	gamelog.BettingAction
	Round int `json:"round"`
}

type persistedPart struct {
	PlayerID int64        `json:"playerId"`
	DidFold  bool         `json:"didFold"`
	Hand     []*deck.Card `json:"hand"`
}

// ParseGameLog decodes a persisted Little L log into a normalized hand.
func ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error) {
	var log persistedLog
	if err := json.Unmarshal(raw, &log); err != nil {
		return nil, fmt.Errorf("could not decode little l log: %w", err)
	}

	hand := &gamelog.Hand{
		AnteCents: log.Ante,
		Rounds:    1,
		Board:     log.Community,
	}

	for _, seat := range log.Seats {
		hand.Participant(seat.PlayerID).StartingCards = seat.StartingHand
	}

	actions := make([]gamelog.BettingAction, 0, len(log.Actions))
	for _, a := range log.Actions {
		a.Street = "round-" + strconv.Itoa(a.Round)
		actions = append(actions, a.BettingAction)
	}
	hand.ApplyBettingActions(actions)

	folded := make(map[int64]bool, len(log.Participants))
	for _, p := range log.Participants {
		folded[p.PlayerID] = p.DidFold
		hand.Participant(p.PlayerID).FinalCards = p.Hand
	}

	// The pot is not stored on the log, so reconstruct it from what was paid out.
	for _, won := range log.Winners {
		hand.PotCents += won
	}

	hand.ResolveShowdown(folded, log.Winners)

	return hand, nil
}
