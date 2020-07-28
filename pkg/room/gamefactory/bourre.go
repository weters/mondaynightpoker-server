package gamefactory

import (
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/bourre"
)

type bourreFactory struct{}

func (b bourreFactory) CreateGame(tableUUID string, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	ante, _ := additionalData.GetInt("ante")
	game, err := bourre.NewGame(tableUUID, playerIDs, bourre.Options{Ante: ante})
	if err != nil {
		return nil, err
	}

	if err := game.Deal(); err != nil {
		return nil, err
	}

	return game, nil
}
