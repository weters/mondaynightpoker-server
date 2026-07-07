package texasholdem

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"
)

// Tick tries to advance the game
func (g *Game) Tick() (bool, error) {
	if g.pendingDealerState != nil {
		if time.Now().After(g.pendingDealerState.After) {
			g.dealerState = g.pendingDealerState.NextState
			g.pendingDealerState = nil

			g.newRoundSetup()

			// don't call new round setup if we are in the pre-flop betting round as are in a good state currently
			// the initial setup was done in the constructor
			if g.dealerState == DealerStatePreFlopBettingRound {
				g.payBlinds()
			}

			return true, nil
		}

		return false, nil
	}

	switch g.dealerState {
	case DealerStateStart:
		if err := g.dealStartingCardsToEachParticipant(); err != nil {
			return false, err
		}

		return true, nil
	case DealerStateDealFlop:
		flop := make([]*deck.Card, 3)
		for i := 0; i < 3; i++ {
			card, err := g.drawCommunityCard()
			if err != nil {
				return false, err
			}

			flop[i] = card
		}

		g.recordStreet("flop", flop...)
		g.SendLogMessage(playable.SimpleLogMessageWithCards(0, flop, "dealer dealt the flop"))
		g.dealerState = DealerStateFlopBettingRound
		return true, nil
	case DealerStateDealTurn:
		card, err := g.drawCommunityCard()
		if err != nil {
			return false, err
		}

		g.recordStreet("turn", card)
		g.SendLogMessages([]*playable.LogMessage{playable.SimpleLogMessageWithCard(0, card, "dealer dealt the turn")})
		g.dealerState = DealerStateTurnBettingRound
		return true, nil
	case DealerStateDealRiver:
		card, err := g.drawCommunityCard()
		if err != nil {
			return false, err
		}

		g.recordStreet("river", card)
		g.SendLogMessages([]*playable.LogMessage{playable.SimpleLogMessageWithCard(0, card, "dealer dealt the river")})
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
	default:
		if g.InBettingRound() && g.potManager.IsRoundOver() {
			g.setPendingDealerState(DealerState(int(g.dealerState)+1), time.Second)
			return true, nil
		}
	}

	return false, nil
}

func (g *Game) drawCommunityCard() (*deck.Card, error) {
	card, err := g.deck.Draw()
	if err != nil {
		return nil, err
	}

	g.community.AddCard(card)
	return card, nil
}
