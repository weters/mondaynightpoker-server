package sevencard

import (
	"mondaynightpoker-server/pkg/deck"
)

// HighChicago is a variant where the player with the highest spade in the hole wins half the pot
type HighChicago struct {
}

// Name returns "High Chicago"
func (h *HighChicago) Name() string {
	return "High Chicago"
}

// Start is a no-op for High Chicago
func (h *HighChicago) Start() {
}

// ParticipantReceivedCard is a no-op for High Chicago
func (h *HighChicago) ParticipantReceivedCard(_ *Game, _ *participant, _ *deck.Card) {
}

// GetSplitPotWinners returns the participant(s) with the highest spade in the hole
// Returns the winning participants, the winning card, and a description for the log message
//
//nolint:revive // participant is intentionally unexported; this method is only called internally via SplitPotVariant interface
func (h *HighChicago) GetSplitPotWinners(g *Game) ([]*participant, *deck.Card, string) {
	var winners []*participant
	var winningCard *deck.Card
	highestRank := 0

	for _, id := range g.playerIDs {
		p := g.idToParticipant[id]
		if p.didFold {
			continue
		}

		// Find highest spade in the hole for this participant
		for _, card := range p.hand {
			// Skip face-up cards - we only care about hole cards
			if card.IsBitSet(faceUp) {
				continue
			}

			if card.Suit == deck.Spades && card.Rank > highestRank {
				highestRank = card.Rank
				winners = []*participant{p}
				winningCard = card
			} else if card.Suit == deck.Spades && card.Rank == highestRank {
				// Tie - add to winners
				winners = append(winners, p)
			}
		}
	}

	if len(winners) == 0 {
		return nil, nil, ""
	}

	return winners, winningCard, "high spade in the hole"
}
