package gamefactory

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/sevencard"
)

type sevenCardFactory struct{}

func (s sevenCardFactory) Details(additionalData playable.AdditionalData) (name string, ante int, err error) {
	opts, err := s.getOptions(additionalData)
	if err != nil {
		return "", 0, err
	}

	return opts.Variant.Name(), opts.Ante, nil
}

func (s sevenCardFactory) CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error) {
	opts, err := s.getOptions(additionalData)
	if err != nil {
		return nil, err
	}

	game, err := sevencard.NewGame(logger, playerIDs, opts)
	if err != nil {
		return nil, err
	}

	if err := game.Start(); err != nil {
		return nil, err
	}

	return game, nil
}

func (s sevenCardFactory) getOptions(additionalData playable.AdditionalData) (sevencard.Options, error) {
	opts := sevencard.DefaultOptions()
	if ante, _ := additionalData.GetInt("ante"); ante > 0 {
		opts.Ante = ante
	}

	if variant, _ := additionalData.GetString("variant"); variant != "" {
		switch variant {
		case "stud":
			opts.Variant = &sevencard.Stud{}
		case "low-card-wild":
			opts.Variant = &sevencard.LowCardWild{}
		case "baseball":
			opts.Variant = &sevencard.Baseball{}
		case "follow-the-queen":
			opts.Variant = &sevencard.FollowTheQueen{}
		default:
			return sevencard.Options{}, fmt.Errorf("unknown seven-card variant: %s", variant)
		}
	}

	return opts, nil
}
