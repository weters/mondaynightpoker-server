package gamefactory

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
)

func Test_gutsFactory_Details(t *testing.T) {
	name, ante, err := factories["guts"].Details(playable.AdditionalData{
		"ante": float64(25),
	})
	assert.NoError(t, err)
	assert.Equal(t, "2-Card Guts", name)
	assert.Equal(t, 25, ante)

	name, ante, err = factories["guts"].Details(playable.AdditionalData{
		"ante": float64(50),
	})
	assert.NoError(t, err)
	assert.Equal(t, "2-Card Guts", name)
	assert.Equal(t, 50, ante)
}

func Test_gutsFactory_CreateGame(t *testing.T) {
	factory := factories["guts"]

	game, err := factory.CreateGame(logrus.StandardLogger(), []int64{1, 2}, playable.AdditionalData{
		"ante": float64(25),
	})
	assert.NoError(t, err)
	assert.NotNil(t, game)
	assert.Equal(t, "guts", game.Name())
}

func Test_gutsFactory_CreateGame_InvalidPlayerCount(t *testing.T) {
	factory := factories["guts"]

	// Too few players
	game, err := factory.CreateGame(logrus.StandardLogger(), []int64{1}, playable.AdditionalData{})
	assert.Error(t, err)
	assert.Nil(t, game)
}

func Test_getGutsOptions(t *testing.T) {
	// Default options
	opts := getGutsOptions(playable.AdditionalData{})
	assert.Equal(t, 25, opts.Ante)
	assert.Equal(t, 1000, opts.MaxOwed)

	// Custom ante
	opts = getGutsOptions(playable.AdditionalData{
		"ante": float64(50),
	})
	assert.Equal(t, 50, opts.Ante)

	// Custom maxOwed within range
	opts = getGutsOptions(playable.AdditionalData{
		"maxOwed": float64(1500),
	})
	assert.Equal(t, 1500, opts.MaxOwed)

	// maxOwed below minimum
	opts = getGutsOptions(playable.AdditionalData{
		"maxOwed": float64(400),
	})
	assert.Equal(t, 1000, opts.MaxOwed) // Should use default

	// maxOwed above maximum
	opts = getGutsOptions(playable.AdditionalData{
		"maxOwed": float64(3000),
	})
	assert.Equal(t, 1000, opts.MaxOwed) // Should use default

	// maxOwed rounded to nearest dollar
	opts = getGutsOptions(playable.AdditionalData{
		"maxOwed": float64(1550),
	})
	assert.Equal(t, 1500, opts.MaxOwed)

	// Zero ante should use default
	opts = getGutsOptions(playable.AdditionalData{
		"ante": float64(0),
	})
	assert.Equal(t, 25, opts.Ante)
}
