package sevencard

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func TestGame_Name(t *testing.T) {
	options := DefaultOptions()
	options.Variant = &Stud{}
	game, _ := NewGame("", []int64{1, 2}, options)
	assert.Equal(t, "Seven-Card Stud", game.Name())
}

func TestGame_LogChan(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2}, DefaultOptions())
	lc := game.LogChan()
	game.logChan <- playable.SimpleLogMessageSlice(0, "test msg")

	msg := <-lc
	assert.Equal(t, "test msg", msg[0].Message)
}

func TestGame_GetPlayerState(t *testing.T) {
	a := assert.New(t)

	game, _ := createTestGame()

	playerState, err := game.GetPlayerState(1)
	a.NoError(err)
	a.NotNil(playerState)
	a.Equal("game", playerState.Key)
	a.Equal("seven-card", playerState.Value)
	a.IsType(PlayerState{}, playerState.Data)
}

func TestGame_Action(t *testing.T) {
	a := assert.New(t)

	game, _ := createTestGame()

	assertActionError := func(playerID int64, message *playable.PayloadIn, expectedErr string) {
		res, updateState, err := game.Action(playerID, message)
		a.Nil(res)
		a.False(updateState)
		a.EqualError(err, expectedErr)
	}

	assertActionOK := func(playerID int64, message *playable.PayloadIn) {
		res, updateState, err := game.Action(playerID, message)
		a.Equal(playable.OK(), res)
		a.True(updateState)
		a.Nil(err)
	}

	payload := func(action string, amount ...int) *playable.PayloadIn {
		var ad playable.AdditionalData
		if len(amount) > 0 {
			ad = playable.AdditionalData{
				"amount": float64(amount[0]),
			}
		}

		return &playable.PayloadIn{
			Action:         action,
			AdditionalData: ad,
		}
	}

	assertActionError(1, payload("bad-action"), "unknown action: bad-action")
	assertActionError(99, payload("check"), "you are not in the game")

	assertActionError(2, payload("check"), "it is not your turn")
	assertActionOK(1, payload("check"))

	assertActionError(2, payload("bet"), "invalid amount")
	assertActionError(1, payload("bet", 25), "it is not your turn")
	assertActionOK(2, payload("bet", 25))

	assertActionError(3, payload("raise"), "invalid amount")
	assertActionError(2, payload("raise", 50), "it is not your turn")
	assertActionOK(3, payload("raise", 50))

	assertActionError(3, payload("fold"), "it is not your turn")
	assertActionOK(1, payload("fold"))

	assertActionError(1, payload("call"), "it is not your turn")
	assertActionOK(2, payload("call"))

	assertActionOK(2, payload("bet", 25))
	assertActionError(3, payload("end-game"), "game is not over")
	assertActionOK(3, payload("fold"))
	assertActionOK(3, payload("end-game"))
}
