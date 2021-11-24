package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"testing"
	"time"
)

type testParticipant struct {
	id         int64
	tableStake int
}

func (t *testParticipant) GetPlayerID() int64 {
	return t.id
}

func (t *testParticipant) GetTableStake() int {
	return t.tableStake
}

func setupParticipants(tableStakes ...int) []playable.Player {
	p := make([]playable.Player, len(tableStakes))
	for i, ts := range tableStakes {
		p[i] = &testParticipant{
			id:         int64(i + 1),
			tableStake: ts,
		}
	}

	return p
}

func setupNewGame(opts Options, tableStakes ...int) *Game {
	game, err := NewGame(logrus.StandardLogger(), setupParticipants(tableStakes...), opts)
	if err != nil {
		panic(err)
	}

	return game
}

func assertAction(t *testing.T, game *Game, playerID int64, action action.Action, msgAndArgs ...interface{}) {
	t.Helper()
	assertActionAndAmount(t, game, playerID, action, 0, msgAndArgs...)
}

func assertActionAndAmount(t *testing.T, game *Game, playerID int64, action action.Action, amount int, msgAndArgs ...interface{}) {
	t.Helper()
	resp, update, err := game.Action(playerID, payload(action, amount))
	assert.NoError(t, err, msgAndArgs...)
	assert.Equal(t, playable.OK(), resp, msgAndArgs...)
	assert.True(t, update, msgAndArgs...)
}

func assertActionFailedAndAmount(t *testing.T, game *Game, playerID int64, action action.Action, amount int, expectedErr string, msgAndArgs ...interface{}) {
	t.Helper()
	resp, update, err := game.Action(playerID, payload(action, amount))
	assert.EqualError(t, err, expectedErr, msgAndArgs...)
	assert.Nil(t, resp, msgAndArgs...)
	assert.False(t, update, msgAndArgs...)
}

func assertActionFailed(t *testing.T, game *Game, playerID int64, action action.Action, expectedErr string, msgAndArgs ...interface{}) {
	t.Helper()
	assertActionFailedAndAmount(t, game, playerID, action, 0, expectedErr, msgAndArgs...)
}

func payload(action action.Action, amount ...int) *playable.PayloadIn {
	amt := 0
	if len(amount) == 1 {
		amt = amount[0]
	}

	return &playable.PayloadIn{
		Action: string(action),
		AdditionalData: playable.AdditionalData{
			"amount": float64(amt),
		},
	}
}

func assertTickFromWaiting(t *testing.T, game *Game, nextState DealerState, msgAndArgs ...interface{}) {
	t.Helper()

	assert.Equal(t, DealerStateWaiting, game.dealerState, msgAndArgs...)
	assert.NotNil(t, game.pendingDealerState, msgAndArgs...)
	game.pendingDealerState.After = time.Now()

	assertTick(t, game, msgAndArgs...)
	assert.Equal(t, nextState, game.dealerState, msgAndArgs...)
}

func assertTick(t *testing.T, game *Game, msgAndArgs ...interface{}) {
	t.Helper()
	update, err := game.Tick()
	assert.NoError(t, err, msgAndArgs...)
	assert.True(t, update, msgAndArgs...)
}
