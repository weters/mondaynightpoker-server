package aceydeucey

import (
	"fmt"
	"strings"
)

// GameType is the Acey Deucey game type
type GameType int

// GameType constants
const (
	GameTypeStandard GameType = iota
	GameTypeContinuousShoe
	GameTypeChaos
)

// String returns the game type
func (g GameType) String() string {
	switch g {
	case GameTypeStandard:
		return "Standard"
	case GameTypeContinuousShoe:
		return "Continuous Shoe"
	case GameTypeChaos:
		return "Chaos"
	}

	panic(fmt.Sprintf("unknown game type: %d", g))
}

// GetGameType returns the GameType based on the string
func GetGameType(s string) (GameType, error) {
	switch strings.ToLower(s) {
	case "standard":
		return GameTypeStandard, nil
	case "continuous shoe":
		return GameTypeContinuousShoe, nil
	case "chaos":
		return GameTypeChaos, nil
	}

	return -1, fmt.Errorf("unknown game type: %s", s)
}

// GetGameTypes returns the game types
func GetGameTypes() map[GameType]string {
	return map[GameType]string{
		GameTypeStandard:       GameTypeStandard.String(),
		GameTypeContinuousShoe: GameTypeContinuousShoe.String(),
		GameTypeChaos:          GameTypeChaos.String(),
	}
}
