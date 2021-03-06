package gamefactory

import (
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/bourre"
)

type bourreFactory struct{}

func (b bourreFactory) Details(additionalData playable.AdditionalData) (string, int, error) {
	ante, _ := additionalData.GetInt("ante")
	return "Bourré", ante, nil
}

func (b bourreFactory) CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	ante, _ := additionalData.GetInt("ante")
	game, err := bourre.NewGame(logger, playerIDs, bourre.Options{Ante: ante})
	if err != nil {
		return nil, err
	}

	if err := game.Deal(); err != nil {
		return nil, err
	}

	return game, nil
}
