package texasholdem

import (
	"fmt"
	"mondaynightpoker-server/pkg/playable"
)

// Action performs a player action
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	panic("implement me")
}

// GetPlayerState returns the current state for the player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	panic("implement me")
}

// GetEndOfGameDetails returns details after the game finishes
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	panic("implement me")
}

// Name returns the name
func (g *Game) Name() string {
	return NameFromOptions(g.options)
}

// NameFromOptions returns the name from the provided options
func NameFromOptions(opts Options) string {
	if err := validateOptions(opts); err != nil {
		return ""
	}

	return fmt.Sprintf("Limit Texas Hold'em (${%d}/${%d})", opts.LowerLimit, opts.LowerLimit*2)
}

// LogChan returns a channel log messages must be sent on
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	panic("implement me")
}
