package deck

import (
	"math"
)

// Hand represents a collection of cards
type Hand []*Card

// AddCard adds a card to the hand
func (h *Hand) AddCard(card *Card) {
	*h = append(*h, card)
}

// HasCard returns true if the hand contains the specified card
func (h *Hand) HasCard(card *Card) bool {
	for _, c := range *h {
		if c.Equal(card) {
			return true
		}
	}

	return false
}

// Discard will discard the specified card
// If max is provided and > 0, then limit to max discards (useful for mega-decks)
func (h *Hand) Discard(card *Card, max ...int) int {
	count := 0
	m := math.MaxInt32
	if len(max) == 1 && max[0] > 0 {
		m = max[0]
	}

	newHand := make([]*Card, 0, len(*h))
	for _, c := range *h {
		if c.Equal(card) && count < m {
			count++
		} else {
			newHand = append(newHand, c)
		}
	}

	*h = newHand
	return count
}
