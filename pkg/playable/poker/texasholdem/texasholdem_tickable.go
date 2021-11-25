package texasholdem

import (
	"github.com/google/uuid"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"
)

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

		g.logChan <- []*playable.LogMessage{{
			UUID:      uuid.New().String(),
			PlayerIDs: nil,
			Cards:     flop,
			Message:   "dealer dealt the flop",
			Time:      time.Now(),
		}}
		g.dealerState = DealerStateFlopBettingRound
		return true, nil
	case DealerStateDealTurn:
		card, err := g.drawCommunityCard()
		if err != nil {
			return false, err
		}

		g.logChan <- []*playable.LogMessage{playable.SimpleLogMessageWithCard(0, card, "dealer dealt the turn")}
		g.dealerState = DealerStateTurnBettingRound
		return true, nil
	case DealerStateDealRiver:
		card, err := g.drawCommunityCard()
		if err != nil {
			return false, err
		}

		g.logChan <- []*playable.LogMessage{playable.SimpleLogMessageWithCard(0, card, "dealer dealt the river")}
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
