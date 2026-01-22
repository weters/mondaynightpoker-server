package gamefactory

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/model"
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

// CreateGame is deprecated, use CreateGameV2 instead
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

// CreateGameV2 creates a new seven-card game with table stake support
func (s sevenCardFactory) CreateGameV2(logger logrus.FieldLogger, players []*model.PlayerTable, additionalData playable.AdditionalData) (playable.Playable, error) {
	opts, err := s.getOptions(additionalData)
	if err != nil {
		return nil, err
	}

	playablePlayers := getPlayersFromPlayerTableList(players)
	game, err := sevencard.NewGameV2(logger, playablePlayers, opts)
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
		case "high-chicago":
			opts.Variant = &sevencard.HighChicago{}
		case "chiggs":
			opts.Variant = &sevencard.Chiggs{}
		default:
			return sevencard.Options{}, fmt.Errorf("unknown seven-card variant: %s", variant)
		}
	}

	return opts, nil
}
