package littlel

import (
	"fmt"
	"mondaynightpoker-server/pkg/playable"
)

// --- Playable Interface ---

// Action performs a game action on behalf of the player
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	switch message.Action {
	case "trade":
	}

	return nil, false, fmt.Errorf("unknown action: %s", message.Action)
}

// GetPlayerState returns the state of the player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	panic("implement me")
}

// GetEndOfGameDetails returns the details at the end of a game
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	panic("implement me")
}

// Name returns the name of the game
func (g *Game) Name() string {
	return "Little L"
}

// LogChan returns a channel that can receive log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}
