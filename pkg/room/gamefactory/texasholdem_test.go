package gamefactory

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/texasholdem"
	"testing"
)

func Test_texasHoldEmFactory_CreateGame(t *testing.T) {
	assert.PanicsWithValue(t, "use CreateGameV2", func() {
		_, _ = factories["texas-hold-em"].CreateGame(logrus.StandardLogger(), []int64{1, 2, 3}, playable.AdditionalData{})
	})
}

func Test_texasHoldEmFactory_CreateGameV2(t *testing.T) {
	a := assert.New(t)

	game, err := factories["texas-hold-em"].(V2).CreateGameV2(logrus.StandardLogger(), []*model.PlayerTable{
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
	a.Equal("Pot-Limit Texas Hold'em (${25}/${50})", name)
	a.Equal(25, ante)

	name, ante, err = factories["texas-hold-em"].Details(playable.AdditionalData{
		"ante":       float64(0),
		"smallBlind": float64(75),
		"bigBlind":   float64(100),
	})
	a.NoError(err)
	a.Equal("Pot-Limit Texas Hold'em (${75}/${100})", name)
	a.Equal(0, ante)
}
