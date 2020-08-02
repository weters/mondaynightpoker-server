package gamefactory

import (
	"fmt"
	"mondaynightpoker-server/pkg/playable"
)

var factories = map[string]GameFactory{
	"bourre":        bourreFactory{},
	"seven-card":    sevenCardFactory{},
	"pass-the-poop": passThePoopFactory{},
	"little-l":      littleLFactory{},
}

// GameFactory is a factory for creating games that implement the Playable interface
type GameFactory interface {
	CreateGame(tableUUID string, playerIDs []int64, additionalData playable.AdditionalData) (playable.Playable, error)
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
