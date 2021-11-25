package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"mondaynightpoker-server/pkg/snapshot"
	"testing"
)

func TestNewGame(t *testing.T) {
	a := assert.New(t)

	runTest := func(t *testing.T, ante, smallBlind, bigBlind, pot int, expectedBalances ...int) {
		t.Helper()

		opts := Options{
			Variant:    Standard,
			Ante:       ante,
			SmallBlind: smallBlind,
			BigBlind:   bigBlind,
		}

		tableStakes := make([]int, len(expectedBalances))
		for i := range expectedBalances {
			tableStakes[i] = 1000
		}

		game, err := NewGame(logrus.StandardLogger(), setupParticipants(tableStakes...), opts)
		a.NoError(err)
		a.NotNil(game)

		// validate small blind and big blind
		a.Equal(pot, game.potManager.GetTotalOnTable(), "pot is good")
		for i, eb := range expectedBalances {
			a.Equal(eb, game.participants[int64(i+1)].balance, "%d balance", i+1)
		}

		// ensure deck is shuffled
		a.NotEqual(deck.New().HashCode(), game.deck.HashCode())

		a.Equal(DealerStateStart, game.dealerState)
	}

	runTest(t, 25, 100, 200, 350, -125, -225)
	runTest(t, 25, 100, 200, 375, -25, -125, -225)
	runTest(t, 50, 50, 100, 300, -50, -100, -150)
}

func TestGame_basicGameWithWinner(t *testing.T) {
	a := assert.New(t)

	// player 1 will have 3C, 4C
	// player 2 will have 3D, 4D
	// player 3 will have 3H, 4H
	// community will be 5h, 6H, 7H, 9S, 10S

	opts := Options{
		Variant:    Standard,
		Ante:       25,
		SmallBlind: 25,
		BigBlind:   50,
	}

	game := setupNewGame(opts, 1000, 1000, 1000)

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
		a.Equal([]action.Action{
			action.Call,
			action.Raise,
			action.Fold,
		}, game.ActionsForParticipant(1))

		_, _, err := game.Action(2, payload(action.Call))
		a.EqualError(err, "you cannot perform call")

		assertAction(t, game, 1, action.Call, "first player can call")
		a.Equal(lastAction{
			Action:   action.Call,
			PlayerID: 1,
		}, *game.lastAction, "last action is correct")

		a.Equal([]action.Action{
			action.Call,
			action.Raise,
			action.Fold,
		}, game.ActionsForParticipant(2))
		a.Nil(game.ActionsForParticipant(1), "nothing to do for 1")
		a.Nil(game.ActionsForParticipant(3), "nothing to do for 3")

		assertActionFailed(t, game, 1, "bad-action", "unknown action for identifier: bad-action", "verify bad action is rejected")
		assertAction(t, game, 2, action.Call, "second player can call")
		a.Equal(lastAction{
			Action:   action.Call,
			PlayerID: 2,
		}, *game.lastAction, "last action is correct")

		a.Equal([]action.Action{
			action.Check,
			action.Raise,
			action.Fold,
		}, game.ActionsForParticipant(3))

		_, _, err = game.Action(2, payload(action.Call))
		a.EqualError(err, "you cannot perform call")

		assertAction(t, game, 3, action.Check, "third player checks to end round")

		assertTickFromWaiting(t, game, DealerStateDealFlop, "state advanced to the flop")
	}

	// in flop
	{
		// verify pot and player balances
		a.Equal(225, game.potManager.GetTotalOnTable())
		a.Equal(-75, game.participants[1].balance)
		a.Equal(0, game.participants[1].bet)
		a.Equal(-75, game.participants[2].balance)
		a.Equal(0, game.participants[2].bet)
		a.Equal(-75, game.participants[3].balance)
		a.Equal(0, game.participants[3].bet)

		assertTick(t, game, "advance state to flop betting round")
		a.Equal(DealerStateFlopBettingRound, game.dealerState, "game is now in the post-flop betting round")
		a.Equal(3, len(game.community), "three cards in the community")

		a.Equal([]action.Action{action.Check, action.Bet, action.Fold}, game.ActionsForParticipant(1))
		a.Nil(game.ActionsForParticipant(2), "only player 1 has actions")
		a.Nil(game.ActionsForParticipant(3), "only player 1 has actions")

		assertAction(t, game, 1, action.Check, "player 1 can check")
		assertAction(t, game, 2, action.Check, "player 2 can check")
		assertActionFailedAndAmount(t, game, 3, action.Bet, 99, "bet must be in increments of ${25}")
		assertActionFailedAndAmount(t, game, 3, action.Bet, 250, "bet must be at most ${225}")
		assertActionAndAmount(t, game, 3, action.Bet, 100, "player 3 can bet") // currentBet = 100
		a.Equal(lastAction{
			Action:   action.Bet,
			Amount:   100,
			PlayerID: 3,
		}, *game.lastAction, "last action is correct")

		a.Equal([]action.Action{action.Call, action.Raise, action.Fold}, game.ActionsForParticipant(1))
		assertActionAndAmount(t, game, 1, action.Raise, 200, "player 1 can raise") // currentBet = 200
		assertActionAndAmount(t, game, 2, action.Raise, 300, "player 2 can raise") // currentBet = 300
		assertAction(t, game, 3, action.Call, "player 2 can call")
		assertActionAndAmount(t, game, 1, action.Raise, 400, "player 1 can raise") // currentBet = 400

		// previously with limit hold'em, we would be capped. ensure that no longer is the case
		a.Equal([]action.Action{action.Call, action.Raise, action.Fold}, game.ActionsForParticipant(2), "ensure we are not capped for raises")

		_, _, err := game.Action(2, payload(action.Raise, game.potManager.GetBet()-25))
		a.EqualError(err, "you cannot raise to an amount less than the current bet")

		assertAction(t, game, 2, action.Call, "player 2 can call")
		assertAction(t, game, 3, action.Call, "player 3 can call")

		assertTickFromWaiting(t, game, DealerStateDealTurn, "state advanced to deal the turn card")
		a.Nil(game.lastAction, "lastAction is reset")

		a.Equal(1425, game.potManager.GetTotalOnTable(), "pot is correct")
		a.Equal(-475, game.participants[1].balance)
		a.Equal(0, game.participants[1].bet)
		a.Equal(-475, game.participants[2].balance)
		a.Equal(0, game.participants[2].bet)
		a.Equal(-475, game.participants[3].balance)
		a.Equal(0, game.participants[3].bet)
	}

	// turn
	{
		assertTick(t, game, "advance to post-turn betting round")
		a.Equal(DealerStateTurnBettingRound, game.dealerState, "now in the turn betting round")
		a.Equal(4, len(game.community), "community has the correct number of cards")
		a.Equal([]action.Action{action.Check, action.Bet, action.Fold}, game.ActionsForParticipant(1), "bet is now 200")
		for i := 1; i <= 3; i++ {
			assertAction(t, game, int64(i), action.Check, "checks all around")
		}

		assertTickFromWaiting(t, game, DealerStateDealRiver, "ready to deal river card")
	}

	// river
	{
		assertTick(t, game, "advance to final betting round")
		a.Equal(DealerStateFinalBettingRound, game.dealerState, "now in the turn betting round")
		a.Equal(5, len(game.community), "community has the correct number of cards")
		a.Equal([]action.Action{action.Check, action.Bet, action.Fold}, game.ActionsForParticipant(1), "bet is still 200")
		for i := 1; i <= 3; i++ {
			assertAction(t, game, int64(i), action.Check, "checks all around")
		}

		assertTickFromWaiting(t, game, DealerStateRevealWinner, "ready to reveal winner")
	}

	// end game
	{
		a.Equal(result(""), game.participants[1].result, "no results yet")
		assertTick(t, game, "advance")

		a.Equal(resultLost, game.participants[1].result)
		a.Equal(0, game.participants[1].winnings)
		a.Equal(-475, game.participants[1].balance)

		a.Equal(resultLost, game.participants[2].result)
		a.Equal(0, game.participants[2].winnings)
		a.Equal(-475, game.participants[2].balance)

		a.Equal(resultWon, game.participants[3].result)
		a.Equal(1425, game.participants[3].winnings)
		a.Equal(950, game.participants[3].balance)

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
			1: -475,
			2: -475,
			3: 950,
		}, details.BalanceAdjustments)
	}
}

func TestGame__multiplePots(t *testing.T) {
	a := assert.New(t)

	opts := Options{
		Variant:    Standard,
		Ante:       0,
		SmallBlind: 25,
		BigBlind:   50,
	}

	game := setupNewGame(opts, 100, 50, 50, 150, 200)

	// Main Pot   = 250  (1,2,3,4,5)
	// Side Pot 1 = 150  (1,4,5)
	// Side Pot 2 = 100  (4,5)

	// Player 1 - BH (Split main, win 2nd) = 275 (175)
	// Player 2 - BH (split main)          = 125 (75)
	// Player 3 - Lose (nada)
	// Player 4 - 2BH  (win 3rd)           = 100 (-50)
	// Player 5 - Lose (nada)

	// deal cards
	assertTick(t, game)
	assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound)

	game.deck.Cards = deck.CardsFromString("2c,4c,6d,8d,10h")
	game.participants[1].cards = deck.CardsFromString("13c,13d")
	game.participants[2].cards = deck.CardsFromString("13h,13s")
	game.participants[3].cards = deck.CardsFromString("11h,11s")
	game.participants[4].cards = deck.CardsFromString("12h,12s")
	game.participants[5].cards = deck.CardsFromString("11h,11s")

	// betting round pre-flop
	{
		assertAction(t, game, 4, action.Call)
		assertAction(t, game, 5, action.Call)
		assertAction(t, game, 1, action.Call)
		assertAction(t, game, 2, action.Call)
		a.Nil(game.ActionsForParticipant(3), "all-in, so no actions")

		assertTickFromWaiting(t, game, DealerStateDealFlop)
		assertTick(t, game)

		a.Equal(1, len(game.potManager.Pots()))
	}

	// flop betting round
	{
		assertActionAndAmount(t, game, 1, action.Bet, 50)
		assertAction(t, game, 4, action.Call)
		assertAction(t, game, 5, action.Call)

		assertTickFromWaiting(t, game, DealerStateDealTurn)
		assertTick(t, game)

		a.Equal(2, len(game.potManager.Pots()))
	}

	// turn betting round
	{
		assertActionAndAmount(t, game, 4, action.Bet, 50)
		assertAction(t, game, 5, action.Call)

		assertTickFromWaiting(t, game, DealerStateDealRiver)
		assertTick(t, game)

		a.Equal(3, len(game.potManager.Pots()))
	}

	// finish game
	{
		assertTick(t, game)
		assertTickFromWaiting(t, game, DealerStateRevealWinner)
		assertTick(t, game)
		assertTickFromWaiting(t, game, DealerStateEnd)
		assertTick(t, game)

		details, isOver := game.GetEndOfGameDetails()
		a.True(isOver)

		a.Equal(map[int64]int{
			1: 175,
			2: 75,
			3: -50,
			4: -50,
			5: -150,
		}, details.BalanceAdjustments)
	}
}

func TestGame_playersFolded(t *testing.T) {
	a := assert.New(t)

	opts := Options{
		Variant:    Standard,
		Ante:       25,
		SmallBlind: 25,
		BigBlind:   50,
	}

	game := setupNewGame(opts, 1000, 1000, 1000)

	// pre-flop
	{
		a.Equal(DealerStateStart, game.dealerState, "at start")
		assertTick(t, game, "game progresses to waiting")
		assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound, "now at pre-flop betting round")

		assertAction(t, game, 1, action.Call)
		assertAction(t, game, 2, action.Call)
		assertAction(t, game, 3, action.Check)

		assertTickFromWaiting(t, game, DealerStateDealFlop)
	}

	// flop
	{
		assertTick(t, game, "move to flop betting round")
		a.Equal(DealerStateFlopBettingRound, game.dealerState)

		assertActionAndAmount(t, game, 1, action.Bet, 50)
		assertAction(t, game, 2, action.Fold)
		assertActionAndAmount(t, game, 3, action.Raise, 100)
		assertAction(t, game, 1, action.Call)

		assertActionFailed(t, game, 2, action.Call, "you cannot perform call", "player folded and thus cannot call")

		assertTickFromWaiting(t, game, DealerStateDealTurn)
	}

	// turn
	{
		assertTick(t, game, "move to turn betting round")
		a.Equal(DealerStateTurnBettingRound, game.dealerState)

		assertAction(t, game, 1, action.Check)
		assertActionAndAmount(t, game, 3, action.Bet, 50)
		assertAction(t, game, 1, action.Fold)

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
	a := assert.New(t)

	opts := Options{
		Variant:    Standard,
		Ante:       25,
		SmallBlind: 25,
		BigBlind:   50,
	}

	game := setupNewGame(opts, 1000, 1000, 1000)

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
		assertAction(t, game, 1, action.Call)
		assertAction(t, game, 2, action.Call)
		assertAction(t, game, 3, action.Check)
		assertTickFromWaiting(t, game, DealerStateDealFlop)
	}

	assertSnapshots(t, game)

	// flop
	{
		assertTick(t, game)
		a.Equal(DealerStateFlopBettingRound, game.dealerState)
		assertAction(t, game, 1, action.Check)
		assertAction(t, game, 2, action.Check)
		assertAction(t, game, 3, action.Check)
		assertTickFromWaiting(t, game, DealerStateDealTurn)
	}

	assertSnapshots(t, game)

	// turn
	{
		assertTick(t, game)
		a.Equal(DealerStateTurnBettingRound, game.dealerState)
		assertAction(t, game, 1, action.Check)
		assertAction(t, game, 2, action.Check)
		assertAction(t, game, 3, action.Check)
		assertTickFromWaiting(t, game, DealerStateDealRiver)
	}

	assertSnapshots(t, game)

	// river
	{
		assertTick(t, game)
		a.Equal(DealerStateFinalBettingRound, game.dealerState)
		assertAction(t, game, 1, action.Check)
		assertAction(t, game, 2, action.Check)
		assertAction(t, game, 3, action.Check)
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
			1: -75,
			2: 50,
			3: 25,
		}, details.BalanceAdjustments)

		a.Equal(resultLost, game.participants[1].result)
		a.Equal(0, game.participants[1].winnings)

		a.Equal(resultWon, game.participants[2].result)
		a.Equal(125, game.participants[2].winnings)

		a.Equal(resultWon, game.participants[3].result)
		a.Equal(100, game.participants[3].winnings)
	}

	assertSnapshots(t, game)
}

func TestGame_firstPlayerFolds(t *testing.T) {
	game := setupNewGame(DefaultOptions(), 1000, 1000, 1000)

	assertTick(t, game)
	assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound, "now at pre-flop betting round")
	assertAction(t, game, 1, action.Fold)
	assertAction(t, game, 2, action.Call)
	assertAction(t, game, 3, action.Check)
	assertTickFromWaiting(t, game, DealerStateDealFlop)
	assertTick(t, game)
	assert.Equal(t, DealerStateFlopBettingRound, game.dealerState)

	turn, err := game.GetCurrentTurn()
	assert.NoError(t, err)
	// fixes a bug previously found
	assert.Equal(t, int64(2), turn.PlayerID, "ensure the first player is skipped")
}

func TestNewGame_withFailures(t *testing.T) {
	a := assert.New(t)

	opts := Options{
		Variant:    Standard,
		Ante:       50,
		BigBlind:   25,
		SmallBlind: 50,
	}
	game, err := NewGame(logrus.StandardLogger(), setupParticipants(100, 100), opts)
	a.EqualError(err, "big blind must be at least ${50}")
	a.Nil(game)

	game, err = NewGame(logrus.StandardLogger(), setupParticipants(100), DefaultOptions())
	a.EqualError(err, "there must be at least two players")
	a.Nil(game)

	game, err = NewGame(logrus.StandardLogger(), setupParticipants(25, 0), DefaultOptions())
	a.EqualError(err, "cannot seat participant without a balance")
	a.Nil(game)
}

func TestGame_dealStartingCardsToEachParticipant__errorStates(t *testing.T) {
	// these errors shouldn't happen
	a := assert.New(t)
	game := setupNewGame(DefaultOptions(), 1000, 1000, 1000)
	game.dealerState = DealerStatePreFlopBettingRound
	a.EqualError(game.dealStartingCardsToEachParticipant(), "cannot deal cards from state 1")

	game.dealerState = DealerStateStart
	game.deck.Cards = deck.CardsFromString("")
	a.EqualError(game.dealStartingCardsToEachParticipant(), "end of deck reached")
}

func TestGame_GetCurrentTurn(t *testing.T) {
	a := assert.New(t)
	game := setupNewGame(DefaultOptions(), 1000, 1000, 1000)

	p, err := game.GetCurrentTurn()
	a.EqualError(err, "not in a betting round")
	a.Nil(p)

	assertTick(t, game)
	assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound)
	p, err = game.GetCurrentTurn()
	a.NoError(err)
	a.Equal(int64(1), p.ID())

	_ = game.potManager.AdvanceDecision()
	_ = game.potManager.AdvanceDecision()
	_ = game.potManager.AdvanceDecision()

	p, err = game.GetCurrentTurn()
	a.EqualError(err, "round is over")
	a.Nil(p)
}

func Test_validateOptions(t *testing.T) {
	a := assert.New(t)
	a.NoError(validateOptions(Options{Variant: Standard}))
	a.EqualError(validateOptions(Options{Variant: Standard, Ante: -1}), "ante must be at least ${0}")
	a.EqualError(validateOptions(Options{Variant: Standard, Ante: 26}), "ante must be in increments of ${25}")
	a.EqualError(validateOptions(Options{Variant: Standard, Ante: 75}), "ante must be at most ${50}")
	a.EqualError(validateOptions(Options{Variant: Standard, SmallBlind: -1}), "small blind must be at least ${0}")
	a.EqualError(validateOptions(Options{Variant: Standard, SmallBlind: 25, BigBlind: 0}), "big blind must be at least ${25}")
}

func assertSnapshots(t *testing.T, game *Game, msgAndArgs ...interface{}) {
	t.Helper()

	for _, p := range game.participantOrder {
		ps, err := game.GetPlayerState(p.ID())
		assert.NoError(t, err, msgAndArgs...)
		snapshot.ValidateSnapshot(t, ps, 1, msgAndArgs...)
	}
}
