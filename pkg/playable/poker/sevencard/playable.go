package sevencard

import "mondaynightpoker-server/pkg/playable"

// Name returns the name of the game
func (g *Game) Name() string {
	return g.options.Variant.Name()
}

// Action performs a game action on behalf of the player
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	panic("implement me")
}

// GetPlayerState returns the player and game state for the specified player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	panic("implement me")
}

// GetEndOfGameDetails returns details about the end of the game if the game is over
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	panic("implement me")
}

// LogChan returns a channel where another goroutine can listen for log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	panic("implement me")
}
