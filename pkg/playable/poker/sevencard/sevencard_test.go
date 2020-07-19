package sevencard

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewGame(t *testing.T) {
	a := assert.New(t)
	game, err := NewGame("", nil, Options{})
	a.EqualError(err, "ante must be greater than zero")
	a.Nil(game)

	game, err = NewGame("", nil, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)
}
