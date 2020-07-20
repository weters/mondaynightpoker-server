package sevencard

import (
	"errors"
	"fmt"
)

func (g *Game) participantFolds(p *participant) error {
	if g.getCurrentTurn() != p {
		return errNotPlayersTurn
	}

	p.didFold = true

	alive := 0
	for _, participant := range g.idToParticipant {
		if !participant.didFold {
			alive++
		}
	}

	switch alive {
	case 1:
		g.endGame()
		return nil
	case 0:
		panic("too many participants folded")
	}

	g.advanceDecision()
	return nil
}

func (g *Game) participantChecks(p *participant) error {
	if g.getCurrentTurn() != p {
		return errNotPlayersTurn
	}

	if g.currentBet > 0 {
		return errors.New("you cannot check with a live bet")
	}

	g.advanceDecision()
	return nil
}

func (g *Game) participantCalls(p *participant) error {
	if g.getCurrentTurn() != p {
		return errNotPlayersTurn
	}

	if g.currentBet == 0 {
		return errors.New("there is no bet to call")
	}

	diff := g.currentBet - p.currentBet
	g.pot += diff
	p.balance -= diff
	g.advanceDecision()
	return nil
}

func (g *Game) participantBets(p *participant, amount int) error {
	if g.currentBet > 0 {
		return errors.New("you must raise with a live bet")
	}

	return g.bet(p, "bet", amount, g.options.Ante)
}

func (g *Game) participantRaises(p *participant, amount int) error {
	if g.currentBet == 0 {
		return errors.New("you cannot raise without a previous bet")
	}

	return g.bet(p, "raise", amount, g.currentBet*2)
}

func (g *Game) bet(p *participant, betType string, amount, min int) error {
	if g.getCurrentTurn() != p {
		return errNotPlayersTurn
	}

	if amount < min {
		return fmt.Errorf("your %s must be at least %d", betType, min)
	}

	if amount%g.options.Ante > 0 {
		return fmt.Errorf("your %s must be divisible by %d", betType, g.options.Ante)
	}

	if amount > g.getMaxBet() {
		return fmt.Errorf("your %s must not exceed %d", betType, g.pot+g.currentBet)
	}

	diff := amount - p.currentBet

	g.pot += diff
	g.currentBet = amount

	p.balance -= diff
	p.currentBet = amount

	g.setDecisionIndexToCurrentTurn()
	g.advanceDecision()
	return nil
}

func (g *Game) participantEndsGame(p *participant) error {
	_ = p

	if !g.isGameOver() {
		return errors.New("game is not over")
	}

	g.done = true
	return nil
}

func (g *Game) getMaxBet() int {
	return g.pot + g.currentBet
}
