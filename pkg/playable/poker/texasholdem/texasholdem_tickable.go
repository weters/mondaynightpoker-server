package texasholdem

import "time"

// Interval returns how often Tick() should be called
func (g *Game) Interval() time.Duration {
	return time.Second
}

// Tick tries to advance the game
func (g *Game) Tick() (bool, error) {
	if g.pendingDealerState != nil {
		if time.Now().After(g.pendingDealerState.After) {
			g.dealerState = g.pendingDealerState.NextState
			g.pendingDealerState = nil

			// don't call new round setup if we are in the pre-flop betting round as are in a good state currently
			// the initial setup was done in the constructor
			if g.dealerState != DealerStatePreFlopBettingRound {
				g.newRoundSetup()
			}

			return true, nil
		}

		return false, nil
	}

	switch g.dealerState {
	case DealerStateStart:
		if err := g.dealTwoCardsToEachParticipant(); err != nil {
			return false, err
		}

		return true, nil
	case DealerStateDealFlop:
		for i := 0; i < 3; i++ {
			if err := g.drawCommunityCard(); err != nil {
				return false, err
			}
		}

		g.dealerState = DealerStateFlopBettingRound
		return true, nil
	case DealerStateDealTurn:
		if err := g.drawCommunityCard(); err != nil {
			return false, err
		}

		g.dealerState = DealerStateTurnBettingRound
		return true, nil
	case DealerStateDealRiver:
		if err := g.drawCommunityCard(); err != nil {
			return false, err
		}

		g.dealerState = DealerStateFinalBettingRound
		return true, nil
	case DealerStateRevealWinner:
		if err := g.endGame(); err != nil {
			return false, err
		}

		return true, nil
	case DealerStateEnd:
		if !g.finished {
			g.finished = true
			return true, nil
		}
	}

	return false, nil
}

func (g *Game) drawCommunityCard() error {
	card, err := g.deck.Draw()
	if err != nil {
		return err
	}

	g.community.AddCard(card)
	return nil
}
