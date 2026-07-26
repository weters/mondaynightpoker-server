package gamefactory

import (
	"encoding/json"
	"fmt"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/gamelog"
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

// GameFactory is the registry entry for a game type. It creates games that
// implement the Playable interface, and also carries the per-game-type knowledge
// that has no other home: the canonical display name and how to read back a
// persisted log. Everything keyed by a game-type identifier lives here so a new
// game cannot be registered with half of it missing.
type GameFactory interface {
	CreateGame(logger logrus.FieldLogger, players []*model.PlayerTable, additionalData playable.AdditionalData) (playable.Playable, error)
	Details(additionalData playable.AdditionalData) (name string, ante int, err error)

	// DisplayName returns the canonical display name for the game type.
	// model.GameTypeGroup(DisplayName()) must yield the same group as every
	// real display name the factory's Details can produce.
	DisplayName() string

	// ParseGameLog decodes the game's persisted games.data payload into a
	// normalized hand. It is the read counterpart to the log each game writes
	// when it ends, and lives on the factory so the mapping from a game-type
	// identifier to the code that understands its log stays in one place.
	//
	// Implementations leave Hand.GameType unset; ParseGameLog fills it in from
	// the identifier it dispatched on.
	ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error)
}

// Get returns a factory by the given name
func Get(name string) (GameFactory, error) {
	factory, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("no factory with name: %s", name)
	}

	return factory, nil
}

// ParseGameLog decodes a persisted games.data payload for the given game type into
// a normalized hand, dispatching to the factory registered under that identifier.
//
// A nil or empty payload yields a nil hand and no error: games are recorded before
// they finish and only get their log on completion, so an unfinished or terminated
// game legitimately has nothing to parse. Callers distinguish that from a failure
// by checking for a nil hand.
func ParseGameLog(gameType string, raw json.RawMessage) (*gamelog.Hand, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	factory, err := Get(gameType)
	if err != nil {
		return nil, err
	}

	hand, err := factory.ParseGameLog(raw)
	if err != nil {
		return nil, err
	}

	if hand != nil {
		hand.GameType = gameType
	}

	return hand, nil
}

// nameByGroup maps a canonical display group to the factory identifier that
// produces it. It is derived from the factories themselves rather than written out
// by hand, relying on the GameFactory contract that
// model.GameTypeGroup(DisplayName()) yields the factory's group.
var nameByGroup = func() map[string]string {
	byGroup := make(map[string]string, len(factories))
	for name, factory := range factories {
		byGroup[model.GameTypeGroup(factory.DisplayName())] = name
	}

	return byGroup
}()

// NameForStoredGameType resolves a games.game_type value to a registered factory
// identifier.
//
// The column stores a game's full display name rather than its identifier, and
// that name varies with the options the game was created with ("4-Card Little L
// (trade: 0, 2)", "Pineapple"). Resolution therefore goes through
// model.GameTypeGroup, which collapses every display name a game can produce down
// to one canonical group.
func NameForStoredGameType(storedGameType string) (string, error) {
	name, ok := nameByGroup[model.GameTypeGroup(storedGameType)]
	if !ok {
		return "", fmt.Errorf("no factory for stored game type: %s", storedGameType)
	}

	return name, nil
}

// ParseStoredGameLog decodes a persisted games.data payload into a normalized hand,
// resolving the games.game_type display name to its factory first. It is the entry
// point for anything reading logs back out of the database.
func ParseStoredGameLog(storedGameType string, raw json.RawMessage) (*gamelog.Hand, error) {
	name, err := NameForStoredGameType(storedGameType)
	if err != nil {
		return nil, err
	}

	return ParseGameLog(name, raw)
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
