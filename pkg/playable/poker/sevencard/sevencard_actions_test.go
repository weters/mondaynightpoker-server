package sevencard

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
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
	a.Equal([]*participant{p(3)}, game.winners)

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

	game, err := NewGame("", []int64{1, 2, 3}, opts)
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

func TestGame_participantEndsGame(t *testing.T) {
	a := assert.New(t)
	game, p := createTestGame()
	a.EqualError(game.participantEndsGame(p(1)), "game is not over")
	a.NoError(game.participantFolds(p(1)))
	a.NoError(game.participantFolds(p(2)))
	a.NoError(game.participantEndsGame(p(1)))
}
