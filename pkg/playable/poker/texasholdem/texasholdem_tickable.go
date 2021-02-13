package texasholdem

import "time"

// Interval returns how often Tick() should be called
func (g *Game) Interval() time.Duration {
	return time.Second
}

// Tick tries to advance the game
func (g *Game) Tick() (bool, error) {
	panic("implement me")
}
