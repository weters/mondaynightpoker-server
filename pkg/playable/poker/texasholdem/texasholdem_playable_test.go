package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGame_Name(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	assert.NoError(t, err)
	assert.Equal(t, "Limit Texas Hold'em (${100}/${200})", g.Name())

	g, err = NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		LowerLimit: 50,
		UpperLimit: 100,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Limit Texas Hold'em (${50}/${100})", g.Name())

	g, err = NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{Ante: -1})
	assert.EqualError(t, err, "ante must be >= ${0}")
	assert.Nil(t, g)
}

func TestGame_getShareOfWinnings(t *testing.T) {
	a := assert.New(t)

	game := &Game{pot: 100}
	a.Equal(100, game.getShareOfWinnings(1, 0))

	a.Equal(50, game.getShareOfWinnings(2, 0))
	a.Equal(50, game.getShareOfWinnings(2, 1))

	a.Equal(50, game.getShareOfWinnings(3, 0))
	a.Equal(25, game.getShareOfWinnings(3, 1))
	a.Equal(25, game.getShareOfWinnings(3, 2))

	game.pot = 125
	a.Equal(50, game.getShareOfWinnings(3, 0))
	a.Equal(50, game.getShareOfWinnings(3, 1))
	a.Equal(25, game.getShareOfWinnings(3, 2))

	game.pot = 350
	a.Equal(125, game.getShareOfWinnings(3, 0))
	a.Equal(125, game.getShareOfWinnings(3, 1))
	a.Equal(100, game.getShareOfWinnings(3, 2))

	a.PanicsWithValue("position is out of range", func() {
		game.getShareOfWinnings(3, 3)
	})
}

func TestGame_Key(t *testing.T) {
	assert.Equal(t, "texas-hold-em", (&Game{}).Key())
}

func TestNameFromOptions(t *testing.T) {
	assert.Equal(t, "", NameFromOptions(Options{Ante: -1}))
}
