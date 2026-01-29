package gamefactory

import (
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/guts"
)

type gutsFactory struct{}

func (g gutsFactory) Details(additionalData playable.AdditionalData) (string, int, error) {
	opts := getGutsOptions(additionalData)
	return guts.NameFromOptions(opts), opts.Ante, nil
}

func (g gutsFactory) CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	opts := getGutsOptions(additionalData)
	game, err := guts.NewGame(logger, playerIDs, opts)
	if err != nil {
		return nil, err
	}

	if err := game.Deal(); err != nil {
		return nil, err
	}

	return game, nil
}

func getGutsOptions(additionalData playable.AdditionalData) guts.Options {
	opts := guts.DefaultOptions()

	if ante, ok := additionalData.GetInt("ante"); ok && ante > 0 {
		opts.Ante = ante
	}

	if maxOwed, ok := additionalData.GetInt("maxOwed"); ok {
		// Validate maxOwed is within acceptable range (500-2500 cents, i.e., $5-$25)
		if maxOwed >= 500 && maxOwed <= 2500 {
			// Round to nearest dollar (100 cents)
			opts.MaxOwed = (maxOwed / 100) * 100
		}
	}

	if cardCount, ok := additionalData.GetInt("cardCount"); ok {
		if cardCount == 2 || cardCount == 3 {
			opts.CardCount = cardCount
		}
	}

	if bloodyGuts, ok := additionalData.GetBool("bloodyGuts"); ok {
		opts.BloodyGuts = bloodyGuts
	}

	return opts
}
