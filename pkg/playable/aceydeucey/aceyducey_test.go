package aceydeucey

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"strconv"
	"testing"
	"time"
)

func TestNewGame(t *testing.T) {
	a := assert.New(t)

	game, err := NewGame(logrus.StandardLogger(), []int64{1}, Options{})
	a.Nil(game)
	a.EqualError(err, "game requires at least two players")

	game, err = NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{})
	a.Nil(game)
	a.EqualError(err, "ante must be > 0")

	game, err = NewGame(logrus.StandardLogger(), []int64{1, 2, 1}, Options{Ante: 25})
	a.Nil(game)
	a.EqualError(err, "duplicate players detected")

	game, err = NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{Ante: 25})
	a.NotNil(game)
	a.NoError(err)

	a.Equal("Acey Deucey", game.Name())
	a.Equal(int64(1), game.participants[1].PlayerID)
	a.Equal(-25, game.participants[1].Balance)
	a.Equal(int64(2), game.participants[2].PlayerID)
	a.Equal(-25, game.participants[2].Balance)
}

func TestAceyDeucey_getCurrentTurn(t *testing.T) {
	a := assert.New(t)
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())
	a.Equal(game.participants[1], game.getCurrentTurn())
	a.Equal(game.participants[1], game.getCurrentTurn())
	game.nextTurn()
	a.Equal(game.participants[2], game.getCurrentTurn())
	game.nextTurn()
	a.Equal(game.participants[3], game.getCurrentTurn())
	game.nextTurn()
	a.Equal(game.participants[1], game.getCurrentTurn())
}

func TestAceyDeucey_isGameOver(t *testing.T) {
	a := assert.New(t)

	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)

	a.False(game.isGameOver())
	game.pot = 0
	a.True(game.isGameOver())
}

func TestAceyDeucey_Key(t *testing.T) {
	g := &Game{}
	assert.Equal(t, "acey-deucey", g.Key())
}

func TestGame_Delay(t *testing.T) {
	g := &Game{}
	assert.Equal(t, time.Second, g.Interval())
}

func TestGame_getCurrentTurn(t *testing.T) {
	a := assert.New(t)
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())
	a.NoError(err)

	a.Equal(game.participants[1], game.getCurrentTurn())
	game.nextTurn()
	a.Equal(game.participants[2], game.getCurrentTurn())
	game.nextTurn()
	a.Equal(game.participants[3], game.getCurrentTurn())
	game.nextTurn()
	a.Equal(game.participants[1], game.getCurrentTurn())
}

func TestGame_basicFlow(t *testing.T) {
	a := assert.New(t)

	opts := DefaultOptions()
	opts.Ante = 100
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	a.NoError(err)

	game.deck.Cards = deck.CardsFromString("14c,3c,2c")
	assertTick(t, game)

	// player 2 is not on the clock
	response, updateState, err := game.Action(2, &playable.PayloadIn{
		Subject: strconv.Itoa(int(ActionPickAceHigh)),
	})
	a.Nil(response)
	a.False(updateState)
	a.EqualError(err, "you cannot perform the action: Pick High Ace")

	assertSuccessfulAction(t, game, 1, ActionPickAceLow, nil)

	assertTick(t, game)
	assertSuccessfulAction(t, game, 1, ActionBet, map[string]interface{}{"amount": float64(25)})
	assertTick(t, game) // now in waiting
	simulateGameWait(game)
	assert.Equal(t, RoundStateRoundOver, game.getCurrentRound().State)

	// player two
	assertTick(t, game)
	assert.Equal(t, 275, game.pot)
	assert.Equal(t, RoundStateStart, game.getCurrentRound().State)
	game.deck.Cards = deck.CardsFromString("2c,4c,4d")
	assertTick(t, game)
	assertTick(t, game)
	assertSuccessfulAction(t, game, 2, ActionBetTheGap, nil)
	assertTick(t, game)
	simulateGameWait(game)
	assert.Equal(t, RoundStateRoundOver, game.getCurrentRound().State)

	// player three
	assertTick(t, game)
	assert.Equal(t, 375, game.pot)
	assert.Equal(t, RoundStateStart, game.getCurrentRound().State)
	game.deck.Cards = deck.CardsFromString("2c,14c,4d")
	assertTick(t, game)
	assertTick(t, game)

	response, updateState, err = game.Action(3, &playable.PayloadIn{
		Subject: strconv.Itoa(int(ActionBet)),
		AdditionalData: playable.AdditionalData{
			"amount": float64(375),
		},
	})
	a.Nil(response)
	a.False(updateState)
	a.EqualError(err, "bet of ${375} exceeds the max bet of ${175}")

	assertSuccessfulAction(t, game, 3, ActionBet, map[string]interface{}{"amount": float64(175)})
	assertTick(t, game)
	simulateGameWait(game)
	assert.Equal(t, RoundStateRoundOver, game.getCurrentRound().State)

	// player 1
	assertTick(t, game)
	assert.Equal(t, 200, game.pot)
	assert.Equal(t, RoundStateStart, game.getCurrentRound().State)
	game.deck.Cards = deck.CardsFromString("2c,14c,4d")
	assertTick(t, game)
	assertTick(t, game)
	assertSuccessfulAction(t, game, 1, ActionBet, map[string]interface{}{"amount": float64(200)})
	assertTick(t, game)
	simulateGameWait(game)
	assert.Equal(t, RoundStateRoundOver, game.getCurrentRound().State)

	assertTick(t, game)
	details, over := game.GetEndOfGameDetails()
	a.Nil(details)
	a.False(over)
	simulateGameWait(game)
	assert.Equal(t, RoundStateComplete, game.getCurrentRound().State)

	details, over = game.GetEndOfGameDetails()
	a.True(over)
	a.Equal(map[int64]int{
		1: 125,
		2: -200,
		3: 75,
	}, details.BalanceAdjustments)

	log := details.Log.([]*Round)
	a.Equal(4, len(log))
}

func assertSuccessfulAction(t *testing.T, game *Game, id int64, action Action, payload map[string]interface{}) {
	resp, didUpdate, err := game.Action(id, &playable.PayloadIn{
		Subject:        strconv.Itoa(int(action)),
		AdditionalData: payload,
	})

	t.Helper()
	assert.Equal(t, playable.OK(), resp)
	assert.True(t, didUpdate)
	assert.NoError(t, err)
}

func assertTick(t *testing.T, game *Game) {
	t.Helper()
	didUpdate, err := game.Tick()
	assert.True(t, didUpdate)
	assert.NoError(t, err)
}

func simulateGameWait(game *Game) {
	game.getCurrentRound().nextAction.Time = time.Time{}
	_, err := game.Tick()
	if err != nil {
		panic(err)
	}
}

func TestGame_newRound_maxBet(t *testing.T) {
	a := assert.New(t)

	opts := DefaultOptions()
	opts.Ante = 100
	g, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)

	a.Equal(150, g.getCurrentRound().getMaxBet())

	// player 2
	g.newRound()
	a.Equal(150, g.getCurrentRound().getMaxBet())

	// player 3
	g.newRound()
	a.Equal(150, g.getCurrentRound().getMaxBet())

	// player 1 - max bet
	g.newRound()
	a.Equal(300, g.getCurrentRound().getMaxBet())
}
