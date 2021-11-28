package texasholdem

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGame_setPendingDealerState(t *testing.T) {
	game := setupNewGame(DefaultOptions(), 100, 100, 100)
	game.setPendingDealerState(DealerStateDealFlop, time.Second)
	assert.NotNil(t, game.pendingDealerState, "pending dealer state set")

	assert.PanicsWithValue(t, "cannot set pending dealer state if one is already present", func() {
		game.setPendingDealerState(DealerStateFinalBettingRound, time.Second)
	})
}

func TestDealerState_String(t *testing.T) {
	a := assert.New(t)
	a.Equal("discard-round", DealerStateDiscardRound.String())
	a.Equal("pre-flop-betting-round", DealerStatePreFlopBettingRound.String())
	a.Equal("deal-flop", DealerStateDealFlop.String())
	a.Equal("flop-betting-round", DealerStateFlopBettingRound.String())
	a.Equal("deal-turn", DealerStateDealTurn.String())
	a.Equal("turn-betting-round", DealerStateTurnBettingRound.String())
	a.Equal("deal-river", DealerStateDealRiver.String())
	a.Equal("final-betting-round", DealerStateFinalBettingRound.String())
	a.Equal("reveal-winner", DealerStateRevealWinner.String())
	a.Equal("end", DealerStateEnd.String())
	a.Equal("waiting", DealerStateWaiting.String())
	a.Equal("", DealerState(-1).String())
}
