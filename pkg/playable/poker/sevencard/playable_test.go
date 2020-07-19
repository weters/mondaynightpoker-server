package sevencard

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestGame_Name(t *testing.T) {
	options := DefaultOptions()
	options.Variant = &Stud{}
	game, _ := NewGame("", []int64{1, 2}, options)
	assert.Equal(t, "Seven-Card Stud", game.Name())
}
