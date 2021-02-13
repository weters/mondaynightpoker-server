package littlel

import (
	"time"
)

// Interval specifies how frequently a Tick() should happen
func (g *Game) Interval() time.Duration {
	return time.Second
}

// Tick is called every Delay() seconds to progress the state of the game
// Currently, this just checks if the round can be ended or if the game can be ended
func (g *Game) Tick() (bool, error) {
	if !g.endGameAt.IsZero() {
		if time.Now().After(g.endGameAt) {
			g.endGameAt = time.Time{}
			g.done = true
			return true, nil
		}

		return false, nil
	}

	if g.IsGameOver() {
		g.endGameAt = time.Now().Add(time.Second * 5)
		return false, nil
	}

	if g.IsRoundOver() {
		if err := g.NextRound(); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}
