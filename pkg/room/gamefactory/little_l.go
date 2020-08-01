package gamefactory

import (
	"fmt"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/littlel"
)

type littleLFactory struct{}

func (l littleLFactory) Name(additionalData playable.AdditionalData) (string, error) {
	opts := getOptions(additionalData)
	tradeIns, err := littlel.NewTradeIns(opts.TradeIns)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Little L (trade: %s)", tradeIns), nil
}

func (l littleLFactory) CreateGame(tableUUID string, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	game, err := littlel.NewGame(tableUUID, playerIDs, getOptions(additionalData))
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
