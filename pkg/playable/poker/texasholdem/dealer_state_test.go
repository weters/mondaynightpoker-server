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
