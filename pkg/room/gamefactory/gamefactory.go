package gamefactory

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
)

var factories = map[string]GameFactory{
	"bourre":        bourreFactory{},
	"seven-card":    sevenCardFactory{},
	"pass-the-poop": passThePoopFactory{},
	"little-l":      littleLFactory{},
	"acey-deucey":   aceyDeuceyFactory{},
	"texas-hold-em": texasHoldEmFactory{},
	"guts":          gutsFactory{},
}

// GameFactory is a factory for creating games that implement the Playable interface
type GameFactory interface {
	CreateGame(logger logrus.FieldLogger, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error)
	Details(additionalData playable.AdditionalData) (name string, ante int, err error)
}

// V2 is a factory for creating games that implement the Playable interface
type V2 interface {
	CreateGameV2(logger logrus.FieldLogger, players []*model.PlayerTable, additionalData playable.AdditionalData) (playable.Playable, error)
	Details(additionalData playable.AdditionalData) (name string, ante int, err error)
}

// Get returns a factory by the given name
func Get(name string) (GameFactory, error) {
	factory, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("no factory with name: %s", name)
	}

	return factory, nil
}

func getPlayersFromPlayerTableList(players []*model.PlayerTable) []playable.Player {
	p := make([]playable.Player, len(players))
	for i, player := range players {
		p[i] = player
	}
	return p
}
