package gamefactory

import (
	"encoding/json"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/aceydeucey"
	"mondaynightpoker-server/pkg/playable/gamelog"

	"github.com/sirupsen/logrus"
)

type aceyDeuceyFactory struct{}

func (a aceyDeuceyFactory) DisplayName() string {
	return "Acey Deucey"
}

func (a aceyDeuceyFactory) CreateGame(logger logrus.FieldLogger, players []*model.PlayerTable, additionalData playable.AdditionalData) (playable.Playable, error) {
	return aceydeucey.NewGameV2(logger, getPlayersFromPlayerTableList(players), getAceyDeuceyOptions(additionalData))
}

func (a aceyDeuceyFactory) Details(additionalData playable.AdditionalData) (name string, ante int, err error) {
	opts := getAceyDeuceyOptions(additionalData)
	return aceydeucey.NameFromOptions(opts), opts.Ante, nil
}

func getAceyDeuceyOptions(data playable.AdditionalData) aceydeucey.Options {
	opts := aceydeucey.DefaultOptions()
	if ante, _ := data.GetInt("ante"); ante > 0 {
		opts.Ante = ante
	}

	if allowPass, ok := data.GetBool("allowPass"); ok {
		opts.AllowPass = allowPass
	}

	if g, _ := data.GetString("gameType"); g != "" {
		if gameType, err := aceydeucey.GetGameType(g); err == nil {
			opts.GameType = gameType
		}
	}

	return opts
}

// ParseGameLog decodes a persisted Acey Deucey log into a normalized hand.
func (a aceyDeuceyFactory) ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error) {
	return aceydeucey.ParseGameLog(raw)
}
