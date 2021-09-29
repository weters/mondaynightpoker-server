package gamefactory

import (
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/bourre"
)

type bourreFactory struct{}

func (b bourreFactory) Details(additionalData playable.AdditionalData) (string, int, error) {
	opts := getBourreOptions(additionalData)
	return bourre.NameFromOptions(opts), opts.Ante, nil
}

func (b bourreFactory) CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	opts := getBourreOptions(additionalData)
	game, err := bourre.NewGame(logger, playerIDs, opts)
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
