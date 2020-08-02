package gamefactory

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/passthepoop"
)

type passThePoopFactory struct{}

func (p passThePoopFactory) Details(additionalData playable.AdditionalData) (string, int, error) {
	opts, err := p.getOptions(additionalData)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("Pass the Poop, %s Edition", opts.Edition.Name()), opts.Ante, nil
}

func (p passThePoopFactory) CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	opts, err := p.getOptions(additionalData)
	if err != nil {
		return nil, err
	}

	game, err := passthepoop.NewGame(logger, playerIDs, opts)
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
