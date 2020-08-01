package gamefactory

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/passthepoop"
)

type passThePoopFactory struct{}

func (p passThePoopFactory) Name(additionalData playable.AdditionalData) (string, error) {
	opts, err := p.getOptions(additionalData)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Pass the Poop, %s Edition", opts.Edition.Name()), nil
}

func (p passThePoopFactory) CreateGame(tableUUID string, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	opts, err := p.getOptions(additionalData)
	if err != nil {
		return nil, err
	}

	game, err := passthepoop.NewGame(tableUUID, playerIDs, opts)
	if err != nil {
		return nil, err
	}

	return game, nil
}

func (p passThePoopFactory) getOptions(additionalData playable.AdditionalData) (passthepoop.Options, error) {
	ante, _ := additionalData.GetInt("ante")
	if ante <= 0 {
		return passthepoop.Options{}, errors.New("ante must be greater than 0")
	}

	edition, _ := additionalData.GetString("edition")
	if edition == "" {
		return passthepoop.Options{}, errors.New("edition is required")
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

	return opts, nil
}
