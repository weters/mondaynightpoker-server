package gamefactory

import (
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/texasholdem"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_texasHoldEmFactory_CreateGame(t *testing.T) {
	a := assert.New(t)

	game, err := factories["texas-hold-em"].CreateGame(logrus.StandardLogger(), []*model.PlayerTable{
		{PlayerID: 1, TableStake: 100},
		{PlayerID: 2, TableStake: 100},
	}, playable.AdditionalData{})
	a.NoError(err)
	a.IsType(&texasholdem.Game{}, game)
}

func Test_texasHoldEmFactory_Details(t *testing.T) {
	a := assert.New(t)
	name, ante, err := factories["texas-hold-em"].Details(playable.AdditionalData{})
	a.NoError(err)
	a.Equal("Texas Hold'em (${25}/${50})", name)
	a.Equal(25, ante)

	name, ante, err = factories["texas-hold-em"].Details(playable.AdditionalData{
		"ante":       float64(0),
		"smallBlind": float64(75),
		"bigBlind":   float64(100),
	})
	a.NoError(err)
	a.Equal("Texas Hold'em (${75}/${100})", name)
	a.Equal(0, ante)
}
