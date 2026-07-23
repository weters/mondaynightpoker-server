package gamefactory

import (
	"fmt"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"sort"

	"github.com/sirupsen/logrus"
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
	CreateGame(logger logrus.FieldLogger, players []*model.PlayerTable, additionalData playable.AdditionalData) (playable.Playable, error)
	Details(additionalData playable.AdditionalData) (name string, ante int, err error)

	// DisplayName returns the canonical display name for the game type.
	// model.GameTypeGroup(DisplayName()) must yield the same group as every
	// real display name the factory's Details can produce.
	DisplayName() string
}

// Get returns a factory by the given name
func Get(name string) (GameFactory, error) {
	factory, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("no factory with name: %s", name)
	}

	return factory, nil
}

// Names returns the registered game-type identifiers, sorted.
func Names() []string {
	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func getPlayersFromPlayerTableList(players []*model.PlayerTable) []playable.Player {
	p := make([]playable.Player, len(players))
	for i, player := range players {
		p[i] = player
	}
	return p
}
