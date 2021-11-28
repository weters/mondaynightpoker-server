package texasholdem

import (
	"encoding/json"
	"time"
)

// DealerState represents the state of the game
type DealerState int

// constants for DealerState
const (
	DealerStateStart DealerState = iota
	DealerStateDiscardRound
	DealerStatePreFlopBettingRound
	DealerStateDealFlop
	DealerStateFlopBettingRound
	DealerStateDealTurn
	DealerStateTurnBettingRound
	DealerStateDealRiver
	DealerStateFinalBettingRound
	DealerStateRevealWinner
	DealerStateEnd
	DealerStateWaiting
)

type pendingDealerState struct {
	NextState DealerState
	After     time.Time
}

func (g *Game) setPendingDealerState(nextState DealerState, after time.Duration) {
	if g.pendingDealerState != nil {
		panic("cannot set pending dealer state if one is already present")
	}

	g.dealerState = DealerStateWaiting
	g.pendingDealerState = &pendingDealerState{
		NextState: nextState,
		After:     time.Now().Add(after),
	}
}

func (d DealerState) String() string {
	switch d {
	case DealerStateStart:
		return "start"
	case DealerStateDiscardRound:
		return "discard-round"
	case DealerStatePreFlopBettingRound:
		return "pre-flop-betting-round"
	case DealerStateDealFlop:
		return "deal-flop"
	case DealerStateFlopBettingRound:
		return "flop-betting-round"
	case DealerStateDealTurn:
		return "deal-turn"
	case DealerStateTurnBettingRound:
		return "turn-betting-round"
	case DealerStateDealRiver:
		return "deal-river"
	case DealerStateFinalBettingRound:
		return "final-betting-round"
	case DealerStateRevealWinner:
		return "reveal-winner"
	case DealerStateEnd:
		return "end"
	case DealerStateWaiting:
		return "waiting"
	}

	return ""
}

// MarshalJSON encodes JSON
func (d DealerState) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		ID:   int(d),
		Name: d.String(),
	})
}
