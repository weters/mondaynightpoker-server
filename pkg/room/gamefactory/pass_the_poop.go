package gamefactory

import (
	"encoding/json"
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/gamelog"
	"mondaynightpoker-server/pkg/playable/passthepoop"

	"github.com/sirupsen/logrus"
)

type passThePoopFactory struct{}

func (p passThePoopFactory) DisplayName() string {
	return "Pass the Poop"
}

func (p passThePoopFactory) Details(additionalData playable.AdditionalData) (string, int, error) {
	opts, err := p.getOptions(additionalData)
	if err != nil {
		return "", 0, err
	}

	name := fmt.Sprintf("Pass the Poop, %s Edition", opts.Edition.Name())

	if opts.AllowBlocks {
		name += " (with Blocks)"
	}

	return name, opts.Ante, nil
}

func (p passThePoopFactory) CreateGame(logger logrus.FieldLogger, players []*model.PlayerTable, additionalData playable.AdditionalData) (playable.Playable, error) {
	opts, err := p.getOptions(additionalData)
	if err != nil {
		return nil, err
	}

	game, err := passthepoop.NewGameV2(logger, getPlayersFromPlayerTableList(players), opts)
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

	allowBlocks, _ := additionalData.GetBool("allowBlocks")
	opts.AllowBlocks = allowBlocks

	return opts, nil
}

// ParseGameLog decodes a persisted Pass the Poop log into a normalized hand.
func (p passThePoopFactory) ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error) {
	return passthepoop.ParseGameLog(raw)
}
