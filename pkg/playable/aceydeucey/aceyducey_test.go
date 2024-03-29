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
	// allowPass isn't true, so ensure user cannot pass
	assertFailedAction(t, game, 1, ActionPass, nil, "you cannot perform the action: Pass")
	assertSuccessfulAction(t, game, 1, ActionBet, betPayload(200))
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

func TestGame_allowPass(t *testing.T) {
	a := assert.New(t)

	opts := Options{
		Ante:      100,
		AllowPass: true,
		GameType:  GameTypeStandard,
	}
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	a.NoError(err)

	game.deck.Cards = deck.CardsFromString("2c,5c")

	// two ticks will deal the first two cards
	assertTick(t, game)
	assertTick(t, game)
	a.Equal(RoundStatePendingBet, game.getCurrentRound().State)

	a.Equal([]Action{ActionPass, ActionBet}, game.getActionsForParticipant(1))
	assertSuccessfulAction(t, game, 1, ActionPass, nil)

	a.Equal(RoundStatePassed, game.getCurrentRound().State)
	assertTick(t, game)
	simulateGameWait(game)
	sg := game.getCurrentRound().Games[game.getCurrentRound().activeGameIndex]
	a.Equal(SingleGameResultPass, sg.Result)
	a.Equal(RoundStateRoundOver, game.getCurrentRound().State)

	// next round
	assertTick(t, game)
	a.Equal(RoundStateStart, game.getCurrentRound().State)

	game.deck.Cards = deck.CardsFromString("2c,5c,3c")
	assertTick(t, game) // deal card 1
	assertTick(t, game) // deal card 2
	assertSuccessfulAction(t, game, 2, ActionBet, betPayload(150))
	assertTick(t, game) // deal card 3
	sg = game.getCurrentRound().Games[game.getCurrentRound().activeGameIndex]
	a.Equal(SingleGameResultWon, sg.Result)

	simulateGameWait(game)
	a.Equal(RoundStateRoundOver, game.getCurrentRound().State) // round over

	// next round
	assertTick(t, game)
	a.Equal(RoundStateStart, game.getCurrentRound().State)

	a.Equal(150, game.pot)
	a.Equal(-100, game.participants[1].Balance) // passed
	a.Equal(50, game.participants[2].Balance)   // won $1.50
	a.Equal(-100, game.participants[3].Balance) // didn't play yet
}

func assertFailedAction(t *testing.T, game *Game, id int64, action Action, payload map[string]interface{}, expectedErr string) {
	resp, didUpdate, err := game.Action(id, &playable.PayloadIn{
		Subject:        strconv.Itoa(int(action)),
		AdditionalData: payload,
	})

	t.Helper()
	assert.EqualError(t, err, expectedErr)
	assert.Nil(t, resp)
	assert.False(t, didUpdate)
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

func betPayload(amount int) map[string]interface{} {
	return map[string]interface{}{
		"amount": float64(amount),
	}
}

func TestGame_newRound(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()

	// test no continuous shoe
	{
		game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)

		a.NoError(err)
		a.NotNil(game)

		// ensure the deck wasn't shuffled when newRound was called
		hc := game.deck.HashCode()
		game.newRound()
		a.Equal(hc, game.getCurrentRound().deck.HashCode())
	}

	// test continuous shoe
	{
		opts.GameType = GameTypeContinuousShoe
		game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
		a.NoError(err)
		a.NotNil(game)

		// ensure the deck is shuffled when newRound is called
		hc := game.deck.HashCode()
		game.newRound()
		a.NotEqual(hc, game.getCurrentRound().deck.HashCode())
	}
}

func TestGame_Name(t *testing.T) {
	game := &Game{}

	a := assert.New(t)
	a.Equal("Acey Deucey", game.Name())

	game.options.GameType = GameTypeContinuousShoe
	a.Equal("Acey Deucey (Continuous Shoe)", game.Name())

	game.options.AllowPass = true
	a.Equal("Acey Deucey (Continuous Shoe and With Passing)", game.Name())

	game.options.GameType = GameTypeStandard
	a.Equal("Acey Deucey (With Passing)", game.Name())
}
