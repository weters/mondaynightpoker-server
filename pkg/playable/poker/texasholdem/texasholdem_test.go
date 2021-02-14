package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"testing"
	"time"
)

func TestNewGame(t *testing.T) {
	a := assert.New(t)

	runTest := func(t *testing.T, ante, lower, upper, p1Bal, p2Bal, p3Bal, pot int) {
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
		a.Equal(lower, game.currentBet, "current bet")
		a.Equal(p1Bal, game.participants[1].Balance, "player 1 balance")
		a.Equal(-1*game.participants[1].Balance-ante, game.participants[1].bet, "player 1 bet")
		a.Equal(p2Bal, game.participants[2].Balance, "player 2 balance")
		a.Equal(-1*game.participants[2].Balance-ante, game.participants[2].bet, "player 2 bet")
		a.Equal(p3Bal, game.participants[3].Balance, "player 3 balance")
		a.Equal(0, game.participants[3].bet, "player 3 bet")
		a.Equal(pot, game.pot, "pot")

		// ensure deck is shuffled
		a.NotEqual(deck.New().HashCode(), game.deck.HashCode())

		a.Equal(DealerStateStart, game.dealerState)
	}

	runTest(t, 25, 100, 200, -75, -125, -25, 225)
	runTest(t, 25, 75, 200, -50, -100, -25, 175)
	runTest(t, 25, 50, 100, -50, -75, -25, 150)
	runTest(t, 25, 25, 50, -50, -50, -25, 125)
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

	g.newRoundSetup()
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
	assertBetAmount(t, DealerStateTurnBettingRound, 200, "")
	assertBetAmount(t, DealerStateFinalBettingRound, 200, "")

	game.currentBet = 600
	assert.True(t, game.CanBet())
	game.currentBet = 800
	assert.False(t, game.CanBet())

	assertBetAmount(t, DealerStateStart, 0, "not in a betting round")
	game.currentBet = 0
	assert.False(t, game.CanBet())
}

func TestGame_game1(t *testing.T) {
	a := assert.New(t)

	// player 1 will have 3C, 4C
	// player 2 will have 3D, 4D
	// player 3 will have 3H, 4H
	// community will be 5h, 6H, 7H, 9S, 10S

	opts := Options{
		Ante:       25,
		LowerLimit: 100,
		UpperLimit: 200,
	}

	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)

	a.NoError(err)
	a.NotNil(game)

	// validate start of game
	{
		a.Equal(DealerStateStart, game.dealerState, "at start of the game")
		a.Equal("", deck.CardsToString(game.participants[1].cards), "no cards have been dealt yet")

		assertTick(t, game, "advance to deal cards")
	}

	// validate initial deal
	{
		a.Equal(2, len(game.participants[1].cards), "has two cards")
		a.Equal(2, len(game.participants[2].cards), "has two cards")
		a.Equal(2, len(game.participants[3].cards), "has two cards")
		assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound, "advance to betting round")

		// setup game
		game.participants[1].cards = deck.CardsFromString("2c,3c,4c")
		game.participants[2].cards = deck.CardsFromString("2d,3d,4d")
		game.participants[3].cards = deck.CardsFromString("2h,3h,4h")
		game.deck.Cards = deck.CardsFromString("5h,6h,7h,9s,10s")
	}

	// validate pre-flop
	{
		a.Nil(game.ActionsForParticipant(1), "no actions for player 1 as third player is first to act")
		a.Nil(game.ActionsForParticipant(2), "no actions for player 2 as third player is first to act")
		a.Equal([]Action{
			{"Call", 100},
			{"Raise", 200},
			actionFold,
		}, game.ActionsForParticipant(3))

		_, _, err = game.Action(1, payload(callKey))
		a.EqualError(err, "you cannot perform Call")

		assertAction(t, game, 3, callKey, "third player can call")

		a.Equal([]Action{
			{"Call", 50},
			{"Raise", 200},
			actionFold,
		}, game.ActionsForParticipant(1))
		a.Nil(game.ActionsForParticipant(2), "nothing to do for 2")
		a.Nil(game.ActionsForParticipant(3), "nothing to do for 3")

		assertAction(t, game, 1, callKey, "first player can call")

		a.Equal([]Action{
			actionCheck,
			{"Raise", 200},
			actionFold,
		}, game.ActionsForParticipant(2))

		_, _, err = game.Action(2, payload(callKey))
		a.EqualError(err, "you cannot perform Call")

		assertAction(t, game, 2, checkKey, "second player checks to end round")
		assertTickFromWaiting(t, game, DealerStateDealFlop, "state advanced to the flop")
	}

	// in flop
	{
		// verify pot and player balances
		a.Equal(375, game.pot)
		a.Equal(-125, game.participants[1].Balance)
		a.Equal(0, game.participants[1].bet)
		a.Equal(-125, game.participants[2].Balance)
		a.Equal(0, game.participants[2].bet)
		a.Equal(-125, game.participants[3].Balance)
		a.Equal(0, game.participants[3].bet)

		assertTick(t, game, "advance state to flop betting round")
		a.Equal(DealerStateFlopBettingRound, game.dealerState, "game is now in the post-flop betting round")
		a.Equal(3, len(game.community), "three cards in the community")

		a.Equal([]Action{actionCheck, {betKey, 100}, actionFold}, game.ActionsForParticipant(1))
		a.Nil(game.ActionsForParticipant(2), "only player 1 has actions")
		a.Nil(game.ActionsForParticipant(3), "only player 1 has actions")

		assertAction(t, game, 1, checkKey, "player 1 can check")
		assertAction(t, game, 2, checkKey, "player 2 can check")
		assertAction(t, game, 3, betKey, "player 3 can bet") // currentBet = 100
		a.Equal(2, game.decisionStart, "decision start is now on player 3")
		a.Equal(1, game.decisionIndex, "decision index is at 1 since player 3 went")

		a.Equal([]Action{{callKey, 100}, {raiseKey, 200}, actionFold}, game.ActionsForParticipant(1))
		assertAction(t, game, 1, raiseKey, "player 1 can raise") // currentBet = 200
		assertAction(t, game, 2, raiseKey, "player 2 can raise") // currentBet = 300
		assertAction(t, game, 3, callKey, "player 2 can call")
		assertAction(t, game, 1, raiseKey, "player 1 can raise") // currentBet = 400

		a.Equal([]Action{{callKey, 100}, actionFold}, game.ActionsForParticipant(2), "ensure we are now capped for raises")

		_, _, err = game.Action(2, payload(raiseKey))
		a.EqualError(err, "you cannot perform Raise")

		assertAction(t, game, 2, callKey, "player 2 can call")
		assertAction(t, game, 3, callKey, "player 3 can call")

		assertTickFromWaiting(t, game, DealerStateDealTurn, "state advanced to deal the turn card")

		a.Equal(1575, game.pot, "pot is correct")
		a.Equal(-525, game.participants[1].Balance)
		a.Equal(0, game.participants[1].bet)
		a.Equal(-525, game.participants[2].Balance)
		a.Equal(0, game.participants[2].bet)
		a.Equal(-525, game.participants[3].Balance)
		a.Equal(0, game.participants[3].bet)
	}

	// turn
	{
		assertTick(t, game, "advance to post-turn betting round")
		a.Equal(DealerStateTurnBettingRound, game.dealerState, "now in the turn betting round")
		a.Equal(4, len(game.community), "community has the correct number of cards")
		a.Equal([]Action{actionCheck, {betKey, 200}, actionFold}, game.ActionsForParticipant(1), "bet is now 200")
		for i := 1; i <= 3; i++ {
			assertAction(t, game, int64(i), checkKey, "checks all around")
		}

		assertTickFromWaiting(t, game, DealerStateDealRiver, "ready to deal river card")
	}

	// river
	{
		assertTick(t, game, "advance to final betting round")
		a.Equal(DealerStateFinalBettingRound, game.dealerState, "now in the turn betting round")
		a.Equal(5, len(game.community), "community has the correct number of cards")
		a.Equal([]Action{actionCheck, {betKey, 200}, actionFold}, game.ActionsForParticipant(1), "bet is still 200")
		for i := 1; i <= 3; i++ {
			assertAction(t, game, int64(i), checkKey, "checks all around")
		}

		assertTickFromWaiting(t, game, DealerStateRevealWinner, "ready to reveal winner")
	}

	// end game
	{
		a.Equal(result(""), game.participants[1].result, "no results yet")
		assertTick(t, game, "advance")

		a.Equal(resultLost, game.participants[1].result)
		a.Equal(resultLost, game.participants[2].result)
		a.Equal(resultWon, game.participants[3].result)

		a.Equal(-525, game.participants[1].Balance)
		a.Equal(-525, game.participants[2].Balance)
		a.Equal(1050, game.participants[3].Balance)

		details, over := game.GetEndOfGameDetails()
		a.Nil(details)
		a.False(over)
		assertTickFromWaiting(t, game, DealerStateEnd, "end the game")
	}

	// GetEndOfGameDetails should return now
	{
		// need to tick once more to get details
		details, over := game.GetEndOfGameDetails()
		a.Nil(details)
		a.False(over)

		assertTick(t, game)

		details, over = game.GetEndOfGameDetails()
		a.NotNil(details)
		a.True(over)

		a.Equal(map[int64]int{
			1: -525,
			2: -525,
			3: 1050,
		}, details.BalanceAdjustments)
	}
}

func assertAction(t *testing.T, game *Game, playerID int64, action string, msgAndArgs ...interface{}) {
	t.Helper()
	resp, update, err := game.Action(playerID, payload(action))
	assert.NoError(t, err, msgAndArgs...)
	assert.Equal(t, playable.OK(), resp, msgAndArgs...)
	assert.True(t, update, msgAndArgs...)
}

func payload(action string) *playable.PayloadIn {
	return &playable.PayloadIn{
		Action: action,
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
