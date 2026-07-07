package sevencard

import "time"

// Tick will try to progress the game
func (g *Game) Tick() (bool, error) {
	if !g.setDoneAt.IsZero() {
		if time.Now().After(g.setDoneAt) {
			g.done = true
			return true, nil
		}

		return false, nil
	}

	if g.isGameOver() {
		g.setDoneAt = time.Now().Add(time.Second * 2)
	}

	return false, nil
}
