package gamefactory

import (
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/littlel"
)

type littleLFactory struct{}

func (l littleLFactory) CreateGame(tableUUID string, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
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

	game, err := littlel.NewGame(tableUUID, playerIDs, opts)
	if err != nil {
		return nil, err
	}

	if err := game.DealCards(); err != nil {
		return nil, err
	}

	return game, nil
}
