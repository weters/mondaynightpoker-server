package sevencard

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func TestGame_participantFolds(t *testing.T) {
	a := assert.New(t)

	game, p := createTestGame()
	a.Equal(errNotPlayersTurn, game.participantFolds(p(2)))
	a.NoError(game.participantFolds(p(1)))
	a.True(p(1).didFold)
	a.False(p(2).didFold)
	a.False(p(3).didFold)
	a.False(game.isGameOver())

	a.Equal(errNotPlayersTurn, game.participantFolds(p(1)))
	a.NoError(game.participantFolds(p(2)))
	a.True(p(1).didFold)
	a.True(p(2).didFold)
	a.False(p(3).didFold)
	a.True(game.isGameOver())
	a.Equal(map[*participant]int{p(3): 75}, game.winners)

	game, p = createTestGame()
	p(2).didFold = true
	p(3).didFold = true
	a.PanicsWithValue("too many participants folded", func() {
		_ = game.participantFolds(p(1))
	})
}

// return a test game
func createTestGame() (*Game, func(id int64) *participant) {
	opts := Options{
		Ante:    25,
		Variant: &Stud{},
	}

	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	if err != nil {
		panic(err)
	}

	if err := game.Start(); err != nil {
		panic(err)
	}

	p := func(id int64) *participant {
		return game.idToParticipant[id]
	}

	p(1).hand = deck.CardsFromString("14c,14d,14h")
	p(2).hand = deck.CardsFromString("13c,13d,13h")
	p(3).hand = deck.CardsFromString("12c,12d,12h")

	p(1).hand[2].SetBit(faceUp)
	p(2).hand[2].SetBit(faceUp)
	p(3).hand[2].SetBit(faceUp)

	game.deck.Cards = deck.CardsFromString("14d,13s,2c,3c,4c,5c,6c")
	game.determineFirstToAct()

	return game, p
}

func TestGame_participantChecks(t *testing.T) {
	a := assert.New(t)

	game, p := createTestGame()

	a.Equal(errNotPlayersTurn, game.participantChecks(p(2)))
	a.NoError(game.participantChecks(p(1)))
	a.NoError(game.participantBets(p(2), 25))
	a.EqualError(game.participantChecks(p(3)), "you cannot check with a live bet")
}

func TestGame_participantCalls(t *testing.T) {
	a := assert.New(t)

	game, p := createTestGame()
	a.EqualError(game.participantCalls(p(1)), "there is no bet to call")
	a.NoError(game.participantBets(p(1), 25))
	a.Equal(errNotPlayersTurn, game.participantCalls(p(1)))

	a.NoError(game.participantCalls(p(2)))
	a.Equal(125, game.pot)
	a.Equal(-50, p(2).balance)
	a.Equal(25, p(1).currentBet)
	a.Equal(25, p(2).currentBet)
	a.Equal(0, p(3).currentBet)

	a.NoError(game.participantCalls(p(3)))
	a.Equal(150, game.pot)
	a.Equal(-50, p(3).balance)
}

func TestGame_participantBets(t *testing.T) {
	a := assert.New(t)

	game, p := createTestGame()
	a.Equal(errNotPlayersTurn, game.participantBets(p(2), 25))
	a.EqualError(game.participantBets(p(1), 100), "your bet must not exceed 75")
	a.EqualError(game.participantBets(p(1), 0), "your bet must be at least 25")
	a.EqualError(game.participantBets(p(1), 26), "your bet must be divisible by 25")
	a.NoError(game.participantBets(p(1), 25), "can bet minimum")

	a.Equal(100, game.pot)
	a.Equal(-50, p(1).balance)
	a.Equal(25, p(1).currentBet)
	a.Equal(-25, p(2).balance)
	a.Equal(0, p(2).currentBet)
	a.Equal(-25, p(3).balance)
	a.Equal(0, p(3).currentBet)

	game, p = createTestGame()
	a.NoError(game.participantBets(p(1), 75), "can bet maximum")
	a.EqualError(game.participantBets(p(2), 150), "you must raise with a live bet")

	a.Equal(150, game.pot)
	a.Equal(-100, p(1).balance)
	a.Equal(75, p(1).currentBet)
	a.Equal(-25, p(2).balance)
	a.Equal(0, p(2).currentBet)
	a.Equal(-25, p(3).balance)
	a.Equal(0, p(3).currentBet)
}

// createTestGameWithTableStakes creates a test game where players have specific table stakes
func createTestGameWithTableStakes(stakes []int) (*Game, func(id int64) *participant) {
	opts := Options{
		Ante:    25,
		Variant: &Stud{},
	}

	players := make([]playable.Player, len(stakes))
	for i, stake := range stakes {
		players[i] = &playable.SimplePlayer{ID: int64(i + 1), TableStake: stake}
	}

	game, err := NewGameV2(logrus.StandardLogger(), players, opts)
	if err != nil {
		panic(err)
	}

	if err := game.Start(); err != nil {
		panic(err)
	}

	p := func(id int64) *participant {
		return game.idToParticipant[id]
	}

	p(1).hand = deck.CardsFromString("14c,14d,14h")
	p(2).hand = deck.CardsFromString("13c,13d,13h")
	p(3).hand = deck.CardsFromString("12c,12d,12h")

	p(1).hand[2].SetBit(faceUp)
	p(2).hand[2].SetBit(faceUp)
	p(3).hand[2].SetBit(faceUp)

	game.deck.Cards = deck.CardsFromString("14d,13s,2c,3c,4c,5c,6c,7c,8c,9c,10c,11c,12c")
	game.determineFirstToAct()

	return game, p
}

func TestGame_allInPlayerIsSkipped(t *testing.T) {
	a := assert.New(t)

	// Player 1 has 25 (just enough for ante, will be all-in immediately)
	// Players 2 and 3 have 500
	game, p := createTestGameWithTableStakes([]int{25, 500, 500})

	// Player 1 should be all-in after ante (balance = 25 - 25 = 0)
	a.Equal(0, p(1).Balance())
	a.True(p(1).isAllIn())

	// Player 1 should have no actions since they're all-in
	a.Equal([]Action{}, game.getActionsForParticipant(p(1)))

	// Player 1 is first to act (best hand), but should be skipped
	// The turn should go to the next non-all-in player
	turn := game.getCurrentTurn()
	a.NotNil(turn)
	a.NotEqual(int64(1), turn.PlayerID, "all-in player should be skipped")
}

func TestGame_participantCallsCappedByTableStake(t *testing.T) {
	a := assert.New(t)

	// Player 1 has 75 (25 for ante, 50 remaining)
	// Player 2 has 500
	// Player 3 has 500
	game, p := createTestGameWithTableStakes([]int{75, 500, 500})

	// Starting pot: 3 * 25 = 75
	a.Equal(75, game.pot)
	a.Equal(50, p(1).Balance())
	a.False(p(1).isAllIn())

	// Player 1 bets 50 (all remaining balance)
	a.NoError(game.participantBets(p(1), 50))
	a.Equal(0, p(1).Balance())
	a.True(p(1).isAllIn())
	a.Equal(125, game.pot) // 75 + 50

	// Player 2 raises to 100
	a.NoError(game.participantRaises(p(2), 100))
	a.Equal(225, game.pot) // 125 + 100

	// Player 3 calls
	a.NoError(game.participantCalls(p(3)))
	a.Equal(325, game.pot) // 225 + 100

	// Player 1 is all-in and should be skipped (not asked to call the raise)
	// The round should advance to the next betting round
	a.NotEqual(firstBettingRound, game.round, "round should have advanced past first betting round")
}

func TestGame_participantCallsPartialWhenShort(t *testing.T) {
	a := assert.New(t)

	// Player 1 has 500
	// Player 2 has 50 (25 for ante, 25 remaining)
	// Player 3 has 500
	game, p := createTestGameWithTableStakes([]int{500, 50, 500})

	a.Equal(25, p(2).Balance())

	// Player 1 bets 50
	a.NoError(game.participantBets(p(1), 50))

	// Player 2 calls but can only afford 25 (goes all-in)
	a.NoError(game.participantCalls(p(2)))
	a.Equal(0, p(2).Balance())
	a.True(p(2).isAllIn())
	// currentBet should be 25 (what they could afford), not 50
	a.Equal(25, p(2).currentBet)
}

func TestGame_allInPlayerHasNoFutureActions(t *testing.T) {
	a := assert.New(t)

	game, p := createTestGameWithTableStakes([]int{25, 500, 500})

	// Player 1 is all-in (only had enough for ante)
	a.True(p(1).isAllIn())
	a.Equal([]Action{}, game.getFutureActionsForParticipant(p(1)))
}

func TestGame_allPlayersAllIn_autoAdvancesToEnd(t *testing.T) {
	a := assert.New(t)

	// All three players have exactly 25 (just enough for ante)
	// All will be all-in after ante, game should auto-advance to revealWinner
	opts := Options{
		Ante:    25,
		Variant: &Stud{},
	}

	players := []playable.Player{
		&playable.SimplePlayer{ID: 1, TableStake: 25},
		&playable.SimplePlayer{ID: 2, TableStake: 25},
		&playable.SimplePlayer{ID: 3, TableStake: 25},
	}

	game, err := NewGameV2(logrus.StandardLogger(), players, opts)
	a.NoError(err)

	p := func(id int64) *participant {
		return game.idToParticipant[id]
	}

	a.True(p(1).isAllIn())
	a.True(p(2).isAllIn())
	a.True(p(3).isAllIn())

	// Start should auto-advance through all rounds to endgame
	a.NoError(game.Start())

	// The game should have auto-advanced through all betting rounds to end
	a.True(game.isGameOver())
	a.NotNil(game.winners)

	// Pot should be 75 (3 * 25 ante) and awarded to the winner
	totalWinnings := 0
	for _, amount := range game.winners {
		totalWinnings += amount
	}
	a.Equal(75, totalWinnings)
}

func TestGame_tableStakes_fullGame(t *testing.T) {
	a := assert.New(t)

	// Player 1 has 200, Player 2 has 50, Player 3 has 300
	// Ante is 25, so after ante: P1=175, P2=25, P3=275
	game, p := createTestGameWithTableStakes([]int{200, 50, 300})

	a.Equal(75, game.pot)
	a.Equal(175, p(1).Balance())
	a.Equal(25, p(2).Balance())
	a.Equal(275, p(3).Balance())

	// Round 1: Player 1 bets 25
	a.NoError(game.participantBets(p(1), 25))
	a.Equal(100, game.pot)

	// Player 2 calls but only has 25 remaining (goes all-in)
	a.NoError(game.participantCalls(p(2)))
	a.Equal(0, p(2).Balance())
	a.True(p(2).isAllIn())
	a.Equal(125, game.pot)

	// Player 3 calls (full 25)
	a.NoError(game.participantCalls(p(3)))
	a.Equal(150, game.pot)

	// Should be in second betting round now
	a.Equal(secondBettingRound, game.round)

	// Player 2 should be skipped (all-in) for all remaining rounds
	// Players 1 and 3 check through remaining rounds
	for game.round >= secondBettingRound && game.round <= finalBettingRound {
		turn := game.getCurrentTurn()
		a.NotNil(turn)
		a.NotEqual(int64(2), turn.PlayerID, "all-in player 2 should never have a turn")
		a.NoError(game.participantChecks(turn))
	}

	// Game should be over
	a.True(game.isGameOver())

	// Pot should still be 150 (no additional bets after round 1)
	totalWinnings := 0
	for _, amount := range game.winners {
		totalWinnings += amount
	}
	a.Equal(150, totalWinnings)
}

func TestGame_getCurrentTurn_panicsOnAllIn(t *testing.T) {
	a := assert.New(t)

	game, _ := createTestGameWithTableStakes([]int{25, 500, 500})

	// Force the decision index onto the all-in player
	game.decisionStartIndex = 0
	game.decisionCount = 0

	a.PanicsWithValue("decision is on a player who is all-in", func() {
		game.getCurrentTurn()
	})
}

func TestGame_participantRaises(t *testing.T) {
	a := assert.New(t)

	game, p := createTestGame()
	a.EqualError(game.participantRaises(p(1), 50), "you cannot raise without a previous bet")
	a.NoError(game.participantBets(p(1), 50))
	a.EqualError(game.participantRaises(p(2), 50), "your raise must be at least 100")
	a.EqualError(game.participantRaises(p(2), 200), "your raise must not exceed 175")
	a.EqualError(game.participantRaises(p(2), 174), "your raise must be divisible by 25")
	a.NoError(game.participantRaises(p(2), 100), "can raise minimum")

	a.Equal(225, game.pot)
	a.Equal(-75, p(1).balance)
	a.Equal(50, p(1).currentBet)
	a.Equal(-125, p(2).balance)
	a.Equal(100, p(2).currentBet)
	a.Equal(-25, p(3).balance)
	a.Equal(0, p(3).currentBet)

	game, p = createTestGame()
	a.NoError(game.participantBets(p(1), 50), "can bet maximum")
	a.Equal(errNotPlayersTurn, game.participantRaises(p(1), 150))
	a.NoError(game.participantRaises(p(2), 175), "can raise max")

	a.Equal(300, game.pot)
	a.Equal(-75, p(1).balance)
	a.Equal(50, p(1).currentBet)
	a.Equal(-200, p(2).balance)
	a.Equal(175, p(2).currentBet)
	a.Equal(-25, p(3).balance)
	a.Equal(0, p(3).currentBet)
}
