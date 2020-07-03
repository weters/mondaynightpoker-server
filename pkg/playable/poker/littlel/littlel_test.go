package littlel

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGame_Name(t *testing.T) {
	g := &Game{}
	assert.Equal(t, "Little L", g.Name())
}

func TestGame_CanTrade(t *testing.T) {
	opts := DefaultOptions()
	game, err := New(opts)
	assert.NoError(t, err)
	assert.True(t, game.CanTrade(0))
	assert.False(t, game.CanTrade(1))
	assert.True(t, game.CanTrade(2))
	assert.False(t, game.CanTrade(3))
	assert.False(t, game.CanTrade(4))

	opts.TradeIns = []int{3, 2, 3, 1}
	game, err = New(opts)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, game.GetAllowedTradeIns())
}

func TestNew(t *testing.T) {
	opts := Options{}
	game, err := New(opts)
	assert.EqualError(t, err, "ante must be greater than zero")
	assert.Nil(t, game)

	opts.Ante = 1
	game, err = New(opts)
	assert.EqualError(t, err, "the initial deal must be between 3 and 5 cards")
	assert.Nil(t, game)

	opts.InitialDeal = 4
	opts.TradeIns = []int{8}
	game, err = New(opts)
	assert.EqualError(t, err, "invalid trade-in option: 8")
	assert.Nil(t, game)

	opts.TradeIns = []int{0, 1, 2, 3, 4}
	game, err = New(opts)
	assert.NoError(t, err)
	assert.NotNil(t, game)
}
