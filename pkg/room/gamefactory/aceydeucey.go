package gamefactory

import (
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/aceydeucey"
)

type aceyDeuceyFactory struct{}

func (a aceyDeuceyFactory) CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	return aceydeucey.NewGame(logger, playerIDs, getAceyDeuceyOptions(additionalData))
}

func (a aceyDeuceyFactory) Details(additionalData playable.AdditionalData) (name string, ante int, err error) {
	opts := getAceyDeuceyOptions(additionalData)
	return aceydeucey.NameFromOptions(opts), opts.Ante, nil
}

func getAceyDeuceyOptions(data playable.AdditionalData) aceydeucey.Options {
	opts := aceydeucey.DefaultOptions()
	if ante, _ := data.GetInt("ante"); ante > 0 {
		opts.Ante = ante
	}

	if allowPass, ok := data.GetBool("allowPass"); ok {
		opts.AllowPass = allowPass
	}

	if continuousShoe, ok := data.GetBool("continuousShoe"); ok {
		opts.ContinuousShoe = continuousShoe
	}

	return opts
}
