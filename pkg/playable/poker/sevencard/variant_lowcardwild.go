package sevencard

import (
	"math"
	"mondaynightpoker-server/pkg/deck"
)

// LowCardWild is a game where the lowest card in the hole is wild
type LowCardWild struct {
}

// Start is a no-op
func (l *LowCardWild) Start() {
}

// ParticipantReceivedCard updates wilds based on the lowest hole card
func (l *LowCardWild) ParticipantReceivedCard(_ *Game, p *participant, _ *deck.Card) {
	lowestRank := math.MaxInt32
	for _, card := range p.hand {
		if card.IsBitSet(faceUp) {
			continue
		}

		if card.Rank < lowestRank {
			lowestRank = card.Rank
		}
	}

	for _, card := range p.hand {
		if card.Rank == lowestRank {
			card.IsWild = true
		} else {
			card.IsWild = false
		}

		if card.IsWild && card.IsBitSet(faceUp) {
			card.SetBit(privateWild)
		} else {
			card.UnsetBit(privateWild)
		}
	}
}

// Name returns the name "Low Card Wild"
func (l *LowCardWild) Name() string {
	return "Low Card Wild"
}
