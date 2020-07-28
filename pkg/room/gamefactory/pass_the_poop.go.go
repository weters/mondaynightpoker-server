package gamefactory

import (
	"errors"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/passthepoop"
)

type passThePoopFactory struct{}

func (p passThePoopFactory) CreateGame(tableUUID string, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	ante, _ := additionalData.GetInt("ante")
	if ante <= 0 {
		return nil, errors.New("ante must be greater than 0")
	}

	edition, _ := additionalData.GetString("edition")
	if edition == "" {
		return nil, errors.New("edition is required")
	}

	opts := passthepoop.DefaultOptions()
	opts.Ante = ante
	switch edition {
	case "standard":
		opts.Edition = &passthepoop.StandardEdition{}
	case "diarrhea":
		opts.Edition = &passthepoop.DiarrheaEdition{}
	case "pairs":
		opts.Edition = &passthepoop.PairsEdition{}
	}

	if lives, _ := additionalData.GetInt("lives"); lives > 0 {
		opts.Lives = lives
	}

	game, err := passthepoop.NewGame(tableUUID, playerIDs, opts)
	if err != nil {
		return nil, err
	}

	return game, nil
}
