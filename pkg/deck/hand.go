package deck

import (
	"math"
	"strings"
)

// Hand represents a collection of cards
type Hand []*Card

func (h Hand) Len() int {
	return len(h)
}

func (h Hand) Less(i, j int) bool {
	if cmp := strings.Compare(string(h[i].Suit), string(h[j].Suit)); cmp != 0 {
		return cmp < 0
	}

	return h[i].Rank < h[j].Rank
}

func (h Hand) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

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

// FirstCard returns the first card in the hand or nil if the cards are empty
func (h Hand) FirstCard() *Card {
	if len(h) == 0 {
		return nil
	}

	return h[0]
}

// LastCard returns the last card in the hand or nil if the cards are empty
func (h Hand) LastCard() *Card {
	n := len(h)
	if n == 0 {
		return nil
	}

	return h[n-1]
}

func (h Hand) String() string {
	return CardsToString(h)
}

// Clone returns a clone of the hand
func (h Hand) Clone() Hand {
	h2 := make(Hand, len(h))
	copy(h2, h)

	return h2
}
