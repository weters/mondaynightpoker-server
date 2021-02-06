package passthepoop

import (
	"time"
)

type tickableAction int

const (
	tickableActionNextRound tickableAction = iota
	tickableActionEndGame
)

type pendingTickableAction struct {
	Action tickableAction
	After  time.Time
}

// Interval specifies how often to call Tick()
func (g *Game) Interval() time.Duration {
	return time.Second
}

// Tick will try to move the game forward
func (g *Game) Tick() (bool, error) {
	if g.pendingTickableAction != nil {
		if time.Now().After(g.pendingTickableAction.After) {
			action := g.pendingTickableAction.Action
			g.pendingTickableAction = nil

			switch action {
			case tickableActionNextRound:
				if err := g.nextRound(); err != nil {
					return false, err
				}

				return true, nil
			case tickableActionEndGame:
				g.endGameAck = true
				return true, nil
			}
		}

		return false, nil
	}

	if g.isGameOver() {
		// game over
		g.pendingTickableAction = &pendingTickableAction{
			Action: tickableActionEndGame,
			After:  time.Now().Add(time.Second),
		}

		return false, nil
	}

	if g.isRoundOver() {
		// next round
		g.pendingTickableAction = &pendingTickableAction{
			Action: tickableActionNextRound,
			After:  time.Now().Add(time.Second),
		}

		return false, nil
	}

	if g.getCurrentTurn() == nil {
		// end round
		if err := g.EndRound(); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}
