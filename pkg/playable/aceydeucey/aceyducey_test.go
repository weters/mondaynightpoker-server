package aceydeucey

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
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

	a.Equal("Acey Ducey", game.Name())
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
