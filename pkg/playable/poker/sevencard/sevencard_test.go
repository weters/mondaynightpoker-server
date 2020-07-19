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

	a.False(game.idToParticipant[1].hand[0].BitField&faceUp > 0)
	a.False(game.idToParticipant[1].hand[1].BitField&faceUp > 0)
	a.True(game.idToParticipant[1].hand[2].BitField&faceUp > 0)

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
