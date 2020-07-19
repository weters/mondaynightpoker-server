package sevencard

import "mondaynightpoker-server/pkg/playable"

// Name returns the name of the game
func (g *Game) Name() string {
	return g.options.Variant.Name()
}

func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	panic("implement me")
}

func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	panic("implement me")
}

func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	panic("implement me")
}

func (g *Game) LogChan() <-chan []*playable.LogMessage {
	panic("implement me")
}
