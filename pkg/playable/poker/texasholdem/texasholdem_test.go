package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/snapshot"
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
		a.Equal(p1Bal, game.participants[1].balance, "player 1 balance")
		a.Equal(-1*game.participants[1].balance-ante, game.participants[1].bet, "player 1 bet")
		a.Equal(p2Bal, game.participants[2].balance, "player 2 balance")
		a.Equal(-1*game.participants[2].balance-ante, game.participants[2].bet, "player 2 bet")
		a.Equal(p3Bal, game.participants[3].balance, "player 3 balance")
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

func TestGame_basicGameWithWinner(t *testing.T) {
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
		a.Equal(lastAction{
			Action:   Action{"Call", 0},
			PlayerID: 3,
		}, *game.lastAction, "last action is correct")

		a.Equal([]Action{
			{"Call", 50},
			{"Raise", 200},
			actionFold,
		}, game.ActionsForParticipant(1))
		a.Nil(game.ActionsForParticipant(2), "nothing to do for 2")
		a.Nil(game.ActionsForParticipant(3), "nothing to do for 3")

		assertActionFailed(t, game, 1, "bad-action", "bad-action is not a valid action", "verify bad action is rejected")
		assertAction(t, game, 1, callKey, "first player can call")
		a.Equal(lastAction{
			Action:   Action{"Call", 0},
			PlayerID: 1,
		}, *game.lastAction, "last action is correct")

		a.Equal([]Action{
			actionCheck,
			{"Raise", 200},
			actionFold,
		}, game.ActionsForParticipant(2))

		_, _, err = game.Action(2, payload(callKey))
		a.EqualError(err, "you cannot perform Call")

		assertAction(t, game, 2, checkKey, "second player checks to end round")

		a.Nil(game.ActionsForParticipant(3), "end of round, no actions")

		assertTickFromWaiting(t, game, DealerStateDealFlop, "state advanced to the flop")
	}

	// in flop
	{
		// verify pot and player balances
		a.Equal(375, game.pot)
		a.Equal(-125, game.participants[1].balance)
		a.Equal(0, game.participants[1].bet)
		a.Equal(-125, game.participants[2].balance)
		a.Equal(0, game.participants[2].bet)
		a.Equal(-125, game.participants[3].balance)
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
		a.Equal(lastAction{
			Action:   Action{"Bet", 0},
			PlayerID: 3,
		}, *game.lastAction, "last action is correct")
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
		a.Nil(game.lastAction, "lastAction is reset")

		a.Equal(1575, game.pot, "pot is correct")
		a.Equal(-525, game.participants[1].balance)
		a.Equal(0, game.participants[1].bet)
		a.Equal(-525, game.participants[2].balance)
		a.Equal(0, game.participants[2].bet)
		a.Equal(-525, game.participants[3].balance)
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
		a.Equal(0, game.participants[1].winnings)
		a.Equal(-525, game.participants[1].balance)

		a.Equal(resultLost, game.participants[2].result)
		a.Equal(0, game.participants[2].winnings)
		a.Equal(-525, game.participants[2].balance)

		a.Equal(resultWon, game.participants[3].result)
		a.Equal(1575, game.participants[3].winnings)
		a.Equal(1050, game.participants[3].balance)

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

func TestGame_playersFolded(t *testing.T) {
	opts := Options{
		Ante:       25,
		LowerLimit: 50,
		UpperLimit: 100,
	}

	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)

	a := assert.New(t)
	a.NoError(err)
	a.NotNil(game)

	// pre-flop
	{
		a.Equal(DealerStateStart, game.dealerState, "at start")
		assertTick(t, game, "game progresses to waiting")
		assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound, "now at pre-flop betting round")

		assertAction(t, game, 3, callKey)
		assertAction(t, game, 1, callKey)
		assertAction(t, game, 2, checkKey)

		assertTickFromWaiting(t, game, DealerStateDealFlop)
	}

	// flop
	{
		assertTick(t, game, "move to flop betting round")
		a.Equal(DealerStateFlopBettingRound, game.dealerState)

		assertAction(t, game, 1, betKey)
		assertAction(t, game, 2, foldKey)
		assertAction(t, game, 3, raiseKey)
		assertAction(t, game, 1, callKey)

		assertActionFailed(t, game, 2, callKey, "you cannot perform Call", "player folded and thus cannot call")

		assertTickFromWaiting(t, game, DealerStateDealTurn)
	}

	// turn
	{
		assertTick(t, game, "move to turn betting round")
		a.Equal(DealerStateTurnBettingRound, game.dealerState)

		assertAction(t, game, 1, checkKey)
		assertAction(t, game, 3, betKey)
		assertAction(t, game, 1, foldKey)

		assertTickFromWaiting(t, game, DealerStateRevealWinner, "game should be over")
	}

	// end game
	{
		assertTick(t, game, "advance to state end")
		assertTickFromWaiting(t, game, DealerStateEnd, "game should be over")
		assertTick(t, game, "finish state")
		a.True(game.finished)

		details, _ := game.GetEndOfGameDetails()
		a.Equal(map[int64]int{
			1: -175,
			2: -75,
			3: 250,
		}, details.BalanceAdjustments)
	}
}

func TestGame_endsInTie(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, Options{
		Ante:       25,
		LowerLimit: 100,
	})

	a := assert.New(t)
	a.NoError(err)

	// community = 2c 4d 6h 8s 10c
	// p1        = 2d 4h
	// p2        = 8c 10c
	// p3        = 8d 10d

	game.deck.Cards = deck.CardsFromString("2d,8c,8d," + "4h,10c,10d," + "2c,4d,6h,8s,10c")

	// setup winners
	{
		game.community = deck.CardsFromString("2c,4d,6h,8s,10c")
		game.participants[1].cards = deck.CardsFromString("2d,4h")
		game.participants[2].cards = deck.CardsFromString("8c,10c")
		game.participants[3].cards = deck.CardsFromString("8d,10d")
	}

	assertSnapshots(t, game)

	// pre-flop
	{
		assertTick(t, game, "start game")
		assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound, "now at pre-flop betting round")
		assertSnapshots(t, game, "ensure currentTurn is set")
		assertAction(t, game, 3, callKey)
		assertAction(t, game, 1, callKey)
		assertAction(t, game, 2, checkKey)
		assertTickFromWaiting(t, game, DealerStateDealFlop)
	}

	assertSnapshots(t, game)

	// flop
	{
		assertTick(t, game)
		a.Equal(DealerStateFlopBettingRound, game.dealerState)
		assertAction(t, game, 1, checkKey)
		assertAction(t, game, 2, checkKey)
		assertAction(t, game, 3, checkKey)
		assertTickFromWaiting(t, game, DealerStateDealTurn)
	}

	assertSnapshots(t, game)

	// turn
	{
		assertTick(t, game)
		a.Equal(DealerStateTurnBettingRound, game.dealerState)
		assertAction(t, game, 1, checkKey)
		assertAction(t, game, 2, checkKey)
		assertAction(t, game, 3, checkKey)
		assertTickFromWaiting(t, game, DealerStateDealRiver)
	}

	assertSnapshots(t, game)

	// river
	{
		assertTick(t, game)
		a.Equal(DealerStateFinalBettingRound, game.dealerState)
		assertAction(t, game, 1, checkKey)
		assertAction(t, game, 2, checkKey)
		assertAction(t, game, 3, checkKey)
		assertTickFromWaiting(t, game, DealerStateRevealWinner)
	}

	assertSnapshots(t, game)

	// end game
	{
		assertTick(t, game)
		assertTickFromWaiting(t, game, DealerStateEnd)
		assertTick(t, game)

		details, _ := game.GetEndOfGameDetails()
		a.Equal(map[int64]int{
			1: -125,
			2: 75,
			3: 50,
		}, details.BalanceAdjustments)

		a.Equal(resultLost, game.participants[1].result)
		a.Equal(0, game.participants[1].winnings)

		a.Equal(resultWon, game.participants[2].result)
		a.Equal(200, game.participants[2].winnings)

		a.Equal(resultWon, game.participants[3].result)
		a.Equal(175, game.participants[3].winnings)
	}

	assertSnapshots(t, game)
}

func TestGame_foldCallCheck(t *testing.T) {
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())

	assertTick(t, game)
	assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound, "now at pre-flop betting round")
	assertAction(t, game, 3, foldKey)
	assertAction(t, game, 1, callKey)
	assertAction(t, game, 2, checkKey)
	assertTickFromWaiting(t, game, DealerStateDealFlop)
}

func assertAction(t *testing.T, game *Game, playerID int64, action string, msgAndArgs ...interface{}) {
	t.Helper()
	resp, update, err := game.Action(playerID, payload(action))
	assert.NoError(t, err, msgAndArgs...)
	assert.Equal(t, playable.OK(), resp, msgAndArgs...)
	assert.True(t, update, msgAndArgs...)
}

func assertActionFailed(t *testing.T, game *Game, playerID int64, action, expectedErr string, msgAndArgs ...interface{}) {
	t.Helper()
	resp, update, err := game.Action(playerID, payload(action))
	assert.EqualError(t, err, expectedErr, msgAndArgs...)
	assert.Nil(t, resp, msgAndArgs...)
	assert.False(t, update, msgAndArgs...)
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

func TestGame_nextDecision(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4, 5}, DefaultOptions())
	a := assert.New(t)

	a.NoError(err)
	a.NotNil(game)

	assertDecision := func(t *testing.T, game *Game, start, index int) {
		t.Helper()
		a.Equal(start, game.decisionStart, "decision start is correct")
		a.Equal(index, game.decisionIndex, "decision index is correct")
	}

	game.dealerState = DealerStateFlopBettingRound
	game.newRoundSetup()
	assertDecision(t, game, 0, 0)

	game.nextDecision()
	assertDecision(t, game, 0, 1)

	game.participants[3].folded = true
	game.participants[4].folded = true

	game.nextDecision()
	assertDecision(t, game, 0, 4)

	// test again
	game.participants[1].folded = true
	game.participants[2].folded = true
	game.participants[3].folded = true
	game.participants[4].folded = false
	game.participants[5].folded = false
	game.decisionStart = 4
	game.decisionIndex = 0
	game.nextDecision()
	assertDecision(t, game, 4, 4)
}

func TestNewGame_withFailures(t *testing.T) {
	a := assert.New(t)

	opts := Options{
		Ante:       50,
		LowerLimit: 25,
	}
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	a.EqualError(err, "ante must be less than the lower limit")
	a.Nil(game)

	game, err = NewGame(logrus.StandardLogger(), []int64{1}, DefaultOptions())
	a.EqualError(err, "there must be at least two players")
	a.Nil(game)
}

func TestGame_dealTwoCardsToEachParticipant_errorStates(t *testing.T) {
	// these errors shouldn't happen
	a := assert.New(t)
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())
	game.dealerState = DealerStatePreFlopBettingRound
	a.EqualError(game.dealTwoCardsToEachParticipant(), "cannot deal cards from state 1")

	game.dealerState = DealerStateStart
	game.deck.Cards = deck.CardsFromString("")
	a.EqualError(game.dealTwoCardsToEachParticipant(), "end of deck reached")
}

func Test_validateOptions(t *testing.T) {
	a := assert.New(t)
	a.EqualError(validateOptions(Options{Ante: -1}), "ante must be >= 0")
	a.EqualError(validateOptions(Options{Ante: 50, LowerLimit: 25}), "ante must be less than the lower limit")
	a.EqualError(validateOptions(Options{Ante: 51, LowerLimit: 100}), "ante must be divisible by ${25}")
	a.EqualError(validateOptions(Options{Ante: 50, LowerLimit: 101}), "lower limit must be divisible by ${25}")
}

func assertSnapshots(t *testing.T, game *Game, msgAndArgs ...interface{}) {
	t.Helper()

	for _, id := range game.participantOrder {
		ps, err := game.GetPlayerState(id)
		assert.NoError(t, err, msgAndArgs...)
		snapshot.ValidateSnapshot(t, ps, 1, msgAndArgs...)
	}
}
