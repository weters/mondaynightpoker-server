package sevencard

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestNewGame(t *testing.T) {
	a := assert.New(t)
	game, err := NewGame("", nil, Options{})
	a.EqualError(err, "ante must be greater than zero")
	a.Nil(game)

	game, err = NewGame("", nil, DefaultOptions())
	a.EqualError(err, "you must have at least two participants")
	a.Nil(game)

	p := make([]int64, 8)
	game, err = NewGame("", p, DefaultOptions())
	a.EqualError(err, "seven-card allows at most 7 participants")
	a.Nil(game)

	game, err = NewGame("", []int64{1, 2}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)

	game, err = NewGame("", []int64{1, 2, 3, 4, 5, 6, 7}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)
}

func TestGame_Start(t *testing.T) {
	a := assert.New(t)

	game, err := NewGame("", []int64{1, 2}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)

	game.deck.Cards = deck.CardsFromString("2c,3c,4c,5c,6c,7c")

	a.NoError(game.Start())
	a.Equal("2c,4c,6c", game.idToParticipant[1].hand.String())
	a.Equal("3c,5c,7c", game.idToParticipant[2].hand.String())
	a.Equal(1, game.decisionStartIndex)

	a.False(game.idToParticipant[1].hand[0].State&faceUp > 0)
	a.False(game.idToParticipant[1].hand[1].State&faceUp > 0)
	a.True(game.idToParticipant[1].hand[2].State&faceUp > 0)

	a.EqualError(game.Start(), "the game has already started")
}

func TestGame_New_notEnoughCards(t *testing.T) {
	a := assert.New(t)

	game, err := NewGame("", []int64{1, 2}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)

	game.deck.Cards = deck.CardsFromString("2c")

	a.EqualError(game.Start(), "end of deck reached")
}

func TestGame_setFirstToAct(t *testing.T) {
	a := assert.New(t)
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	game.idToParticipant[1].hand = deck.CardsFromString("14c,14c,4c")
	game.idToParticipant[2].hand = deck.CardsFromString("2c,3c,14c")
	game.idToParticipant[3].hand = deck.CardsFromString("14c,14c,14c")

	for _, p := range game.idToParticipant {
		p.hand[2].State |= faceUp
	}

	game.determineFirstToAct()
	a.Equal(1, game.decisionStartIndex)

	// folded players don't count
	game.idToParticipant[2].didFold = true
	game.determineFirstToAct()
	a.Equal(2, game.decisionStartIndex)
}

func TestGame_setFirstToAct_withMoreCards(t *testing.T) {
	a := assert.New(t)
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	game.idToParticipant[1].hand = deck.CardsFromString("14c,14c,4c,5d")
	game.idToParticipant[2].hand = deck.CardsFromString("2c,3c,14c,3c")
	game.idToParticipant[3].hand = deck.CardsFromString("14c,14c,8c,8d")

	for _, p := range game.idToParticipant {
		p.hand[2].State |= faceUp
		p.hand[3].State |= faceUp
	}

	game.determineFirstToAct()
	a.Equal(2, game.decisionStartIndex)
}

func TestGame_turns(t *testing.T) {
	a := assert.New(t)

	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	a.NoError(game.Start())
	game.decisionStartIndex = 0

	a.Equal(int64(1), game.getCurrentTurn().PlayerID)
	game.advanceDecisionIfPlayerDidFold() // no movement
	a.Equal(int64(1), game.getCurrentTurn().PlayerID)

	game.advanceDecision()
	a.Equal(int64(2), game.getCurrentTurn().PlayerID)

	game.advanceDecision()
	a.Equal(int64(3), game.getCurrentTurn().PlayerID)

	game.advanceDecision()
	a.Nil(game.getCurrentTurn())

	// reset to test for folds
	game.decisionStartIndex = 0
	game.decisionCount = 0
	a.Equal(int64(1), game.getCurrentTurn().PlayerID) // ensure good state
	game.idToParticipant[1].didFold = true
	game.idToParticipant[2].didFold = true
	game.advanceDecisionIfPlayerDidFold()
	a.Equal(int64(3), game.getCurrentTurn().PlayerID)

	// test what happens if we make a coding error and don't advance beyond a folded player
	game.decisionCount = 0
	a.PanicsWithValue("decision is on a player who folded", func() {
		game.getCurrentTurn()
	})

	// test end of game
	game.round = finalBettingRound + 1
	a.Nil(game.getCurrentTurn(), "no turn if we finished game")
}
