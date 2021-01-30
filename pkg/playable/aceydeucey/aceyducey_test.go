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

	game, err = NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{Ante: 25})
	a.NotNil(game)
	a.NoError(err)

	a.Equal("Acey Ducey", game.Name())
	a.Equal(int64(1), game.participants[1].playerID)
	a.Equal(-25, game.participants[1].balance)
	a.Equal(int64(2), game.participants[2].playerID)
	a.Equal(-25, game.participants[2].balance)
}
