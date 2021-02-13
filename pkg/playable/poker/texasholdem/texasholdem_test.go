package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestNewGame(t *testing.T) {
	a := assert.New(t)

	runTest := func(t *testing.T, ante, lower, upper, p1Bal, p2Bal, p3Bal int) {
		t.Helper()

		opts := Options{
			Ante:       ante,
			LowerLimit: lower,
			UpperLimit: upper,
		}

		game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
		a.NoError(err)
		a.NotNil(game)

		// validate small blind and big blind
		a.Equal(p1Bal, game.participants[1].Balance, "player 1 balance")
		a.Equal(p2Bal, game.participants[2].Balance, "player 2 balance")
		a.Equal(p3Bal, game.participants[3].Balance, "player 3 balance")

		// ensure deck is shuffled
		a.NotEqual(deck.New().HashCode(), game.deck.HashCode())

		a.Equal(DealerStateStart, game.dealerState)
	}

	runTest(t, 25, 100, 200, -75, -125, -25)
	runTest(t, 25, 75, 200, -50, -100, -25)
	runTest(t, 25, 50, 100, -50, -75, -25)
	runTest(t, 25, 25, 50, -50, -50, -25)
}

func TestGame_GetCurrentTurn(t *testing.T) {
	a := assert.New(t)
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, DefaultOptions())
	a.NoError(err)
	a.NotNil(g)

	turn, err := g.GetCurrentTurn()
	a.EqualError(err, "not in a betting round")
	a.Nil(turn)

	assertCurrentTurn := func(t *testing.T, id int64) {
		t.Helper()

		turn, err := g.GetCurrentTurn()
		assert.NoError(t, err)
		assert.Equal(t, id, turn.PlayerID)
	}

	g.dealerState = DealerStateFinalBettingRound
	assertCurrentTurn(t, 1)

	g.decisionIndex++
	assertCurrentTurn(t, 2)

	g.decisionStart = 3
	g.decisionIndex = 2
	assertCurrentTurn(t, 2)

	g.decisionIndex = 4
	turn, err = g.GetCurrentTurn()
	a.EqualError(err, "betting round is over")
	a.Nil(turn)
}

func TestGame_GetBetAmount(t *testing.T) {
	opts := Options{
		LowerLimit: 100,
		UpperLimit: 200,
	}

	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	game.dealerState = DealerStatePreFlopBettingRound

	assertBetAmount := func(t *testing.T, ds DealerState, expectedAmt int, expectedErr string) {
		t.Helper()

		game.dealerState = ds
		amt, err := game.GetBetAmount()
		if expectedErr == "" {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, expectedErr)
		}

		assert.Equal(t, expectedAmt, amt)
	}

	assertBetAmount(t, DealerStatePreFlopBettingRound, 100, "")
	assert.True(t, game.CanBet())
	game.currentBet = 300
	assert.True(t, game.CanBet())
	game.currentBet = 400
	assert.False(t, game.CanBet())

	assertBetAmount(t, DealerStateFlopBettingRound, 100, "")
	assertBetAmount(t, DealerStateRiverBettingRound, 200, "")
	assertBetAmount(t, DealerStateFinalBettingRound, 200, "")

	game.currentBet = 600
	assert.True(t, game.CanBet())
	game.currentBet = 800
	assert.False(t, game.CanBet())

	assertBetAmount(t, DealerStateStart, 0, "not in a betting round")
	game.currentBet = 0
	assert.False(t, game.CanBet())
}
