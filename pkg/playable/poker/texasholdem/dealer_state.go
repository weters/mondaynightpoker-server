package texasholdem

import "time"

// DealerState represents the state of the game
type DealerState int

// constants for DealerState
const (
	DealerStateStart DealerState = iota
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
