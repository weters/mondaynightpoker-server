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
	InitialDeal  int             `json:"initialDeal"`
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

// persistedAct carries Round as an integer because littlel's round type is a plain
// int with no custom marshalling, unlike the named streets the other poker
// variants record.
type persistedAct struct {
	Round    int           `json:"round"`
	PlayerID int64         `json:"playerId"`
	Action   gamelog.RawID `json:"action"`
	Amount   int           `json:"amount"`
	Cards    []*deck.Card  `json:"cards"`
	AllIn    bool          `json:"allIn"`
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
		actions = append(actions, gamelog.BettingAction{
			Street:   "round-" + strconv.Itoa(a.Round),
			PlayerID: a.PlayerID,
			Action:   a.Action,
			Amount:   a.Amount,
			Cards:    a.Cards,
			AllIn:    a.AllIn,
		})
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
