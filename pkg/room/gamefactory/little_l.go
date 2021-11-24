package gamefactory

import (
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/littlel"
)

type littleLFactory struct{}

func (l littleLFactory) Details(additionalData playable.AdditionalData) (string, int, error) {
	opts := getOptions(additionalData)
	name, err := littlel.NameFromOptions(opts)
	if err != nil {
		return "", 0, err
	}

	return name, opts.Ante, nil
}

func (l littleLFactory) CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	panic("use CreateGameV2")
}

func (l littleLFactory) CreateGameV2(logger logrus.FieldLogger, players []*model.PlayerTable, additionalData playable.AdditionalData) (playable.Playable, error) {
	p := getPlayersFromPlayerTableList(players)

	game, err := littlel.NewGameV2(logger, p, getOptions(additionalData))
	if err != nil {
		return nil, err
	}

	if err := game.DealCards(); err != nil {
		return nil, err
	}

	return game, nil
}

func getOptions(additionalData playable.AdditionalData) littlel.Options {
	opts := littlel.DefaultOptions()
	if ante, _ := additionalData.GetInt("ante"); ante > 0 {
		opts.Ante = ante
	}

	if initialDeal, _ := additionalData.GetInt("initialDeal"); initialDeal > 0 {
		opts.InitialDeal = initialDeal
	}

	if tradeIns, ok := additionalData.GetIntSlice("tradeIns"); ok {
		opts.TradeIns = tradeIns
	}

	return opts
}
