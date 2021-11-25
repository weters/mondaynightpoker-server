package gamefactory

import (
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/texasholdem"
)

type texasHoldEmFactory struct{}

func (t texasHoldEmFactory) CreateGameV2(logger logrus.FieldLogger, players []*model.PlayerTable, additionalData playable.AdditionalData) (playable.Playable, error) {
	p := getPlayersFromPlayerTableList(players)
	return texasholdem.NewGame(logger, p, texasHoldEmOptions(additionalData))
}

func (t texasHoldEmFactory) CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	panic("use CreateGameV2")
}

func (t texasHoldEmFactory) Details(additionalData playable.AdditionalData) (name string, ante int, err error) {
	opts := texasHoldEmOptions(additionalData)
	name = texasholdem.NameFromOptions(opts)

	return name, opts.Ante, nil
}

func texasHoldEmOptions(additionData playable.AdditionalData) texasholdem.Options {
	opts := texasholdem.DefaultOptions()

	if variantStr, _ := additionData.GetString("variant"); variantStr != "" {
		if variant, err := texasholdem.VariantFromString(variantStr); err != nil {
			logrus.WithError(err).Error("invalid variant")
		} else {
			opts.Variant = variant
		}
	}

	if ante, ok := additionData.GetInt("ante"); ok && ante >= 0 {
		opts.Ante = ante
	}

	if smallBlind, ok := additionData.GetInt("smallBlind"); ok && smallBlind >= 0 {
		opts.SmallBlind = smallBlind
	}

	if bigBlind, ok := additionData.GetInt("bigBlind"); ok && bigBlind >= 0 {
		opts.BigBlind = bigBlind
	}

	return opts
}
