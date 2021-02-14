package gamefactory

import (
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/texasholdem"
)

type texasHoldEmFactory struct{}

func (t texasHoldEmFactory) CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	return texasholdem.NewGame(logger, playerIDs, texasholdemOptions(additionalData))
}

func (t texasHoldEmFactory) Details(additionalData playable.AdditionalData) (name string, ante int, err error) {
	opts := texasholdemOptions(additionalData)
	name = texasholdem.NameFromOptions(opts)

	return name, opts.Ante, nil
}

func texasholdemOptions(additionData playable.AdditionalData) texasholdem.Options {
	opts := texasholdem.DefaultOptions()

	if ante, _ := additionData.GetInt("ante"); ante > 0 {
		opts.Ante = ante
	}

	if lowerLimit, _ := additionData.GetInt("lowLimit"); lowerLimit > 0 {
		opts.LowerLimit = lowerLimit
	}

	return opts
}
