package gamefactory

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/texasholdem"
	"testing"
)

func Test_texasHoldEmFactory_CreateGame(t *testing.T) {
	a := assert.New(t)
	game, err := factories["texas-hold-em"].CreateGame(logrus.StandardLogger(), []int64{1, 2, 3}, playable.AdditionalData{})
	a.NoError(err)
	a.IsType(&texasholdem.Game{}, game)
}

func Test_texasHoldEmFactory_Details(t *testing.T) {
	a := assert.New(t)
	name, ante, err := factories["texas-hold-em"].Details(playable.AdditionalData{})
	a.NoError(err)
	a.Equal("Limit Texas Hold'em (${100}/${200})", name)
	a.Equal(25, ante)

	name, ante, err = factories["texas-hold-em"].Details(playable.AdditionalData{
		"ante":     float64(0),
		"lowLimit": float64(25),
	})
	a.NoError(err)
	a.Equal("Limit Texas Hold'em (${25}/${50})", name)
	a.Equal(0, ante)
}
