package gamefactory

import (
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/bourre"

	"github.com/sirupsen/logrus"
)

type bourreFactory struct{}

func (b bourreFactory) DisplayName() string {
	return "Bourré"
}

func (b bourreFactory) Details(additionalData playable.AdditionalData) (string, int, error) {
	opts := getBourreOptions(additionalData)
	return bourre.NameFromOptions(opts), opts.Ante, nil
}

func (b bourreFactory) CreateGame(logger logrus.FieldLogger, players []*model.PlayerTable, additionalData playable.AdditionalData) (playable.Playable, error) {
	opts := getBourreOptions(additionalData)
	game, err := bourre.NewGameV2(logger, getPlayersFromPlayerTableList(players), opts)
	if err != nil {
		return nil, err
	}

	if err := game.Deal(); err != nil {
		return nil, err
	}

	return game, nil
}

func getBourreOptions(additionalData playable.AdditionalData) bourre.Options {
	opts := bourre.DefaultOptions()
	if ante, _ := additionalData.GetInt("ante"); ante > 0 {
		opts.Ante = ante
	}

	if fiveSuit, _ := additionalData.GetBool("fiveSuit"); fiveSuit {
		opts.FiveSuit = true
	}

	return opts
}
