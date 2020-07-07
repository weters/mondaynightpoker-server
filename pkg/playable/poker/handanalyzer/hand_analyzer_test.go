package handanalyzer

import (
	"fmt"
	"log"
	"mondaynightpoker-server/pkg/deck"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandAnalyzer_GetFourOfAKind(t *testing.T) {
	h := New(5, deck.CardsFromString("2c,3c,3d,3h,3s"))
	r, ok := h.GetFourOfAKind()
	assert.True(t, ok)
	assert.Equal(t, 3, r)
	_, ok = h.GetThreeOfAKind()
	assert.False(t, ok)
	_, ok = h.GetPair()
	assert.False(t, ok)

	h = New(5, deck.CardsFromString("4s,4h,5c,4d,4c"))
	r, ok = h.GetFourOfAKind()
	assert.True(t, ok)
	assert.Equal(t, 4, r)

	h = New(5, deck.CardsFromString("9s,4h,5c,4d,4c"))
	r, ok = h.GetFourOfAKind()
	assert.False(t, ok)
	assert.Equal(t, 0, r)
}

func TestHandAnalyzer_GetFullHouse(t *testing.T) {
	h := New(5, deck.CardsFromString("14c,2c,14d,5c,14h,2d,5h"))
	r, ok := h.GetFullHouse()
	assert.True(t, ok)
	assert.Equal(t, []int{14, 5}, r)

	h = New(5, deck.CardsFromString("3c,3d,3h,4c,4d,4h,5c"))
	r, ok = h.GetFullHouse()
	assert.True(t, ok)
	assert.Equal(t, []int{4, 3}, r)

	// prefer the pair over the second trip
	h = New(5, deck.CardsFromString("3c,3d,3h,4c,4d,4h,5c,5d"))
	r, ok = h.GetFullHouse()
	assert.True(t, ok)
	assert.Equal(t, []int{4, 5}, r)

	// prefer the second trip over the pair
	h = New(5, deck.CardsFromString("7c,7d,7h,6c,6d,6h,5c,5d"))
	r, ok = h.GetFullHouse()
	assert.True(t, ok)
	assert.Equal(t, []int{7, 6}, r)

	h = New(5, deck.CardsFromString("3c,3d,3h,4c,5d,6h,7c"))
	r, ok = h.GetFullHouse()
	assert.False(t, ok)
	assert.Nil(t, r)

	h = New(5, deck.CardsFromString("3c,3d,4h,4c,5d,5h,6c"))
	r, ok = h.GetFullHouse()
	assert.False(t, ok)
	assert.Nil(t, r)
}

func TestHandAnalyzer_GetHighCard(t *testing.T) {
	h := New(5, deck.CardsFromString("14c,2c,5c,8d,3h"))
	r, ok := h.GetHighCard()
	assert.Equal(t, []int{14, 8, 5, 3, 2}, r)
	assert.True(t, ok)
}

func TestHandAnalyzer_GetPair(t *testing.T) {
	h := New(5, deck.CardsFromString("2c,5c,2h,5h,6d"))
	r, ok := h.GetPair()
	assert.True(t, ok)
	assert.Equal(t, 5, r)

	h = New(5, deck.CardsFromString("2c,3c,4h,5h,6d"))
	r, ok = h.GetPair()
	assert.False(t, ok)
	assert.Equal(t, 0, r)
}

func TestHandAnalyzer_GetTrips(t *testing.T) {
	h := New(5, deck.CardsFromString("2c,5c,5h,5h,6d,4c,4d,4h"))
	r, ok := h.GetThreeOfAKind()
	assert.True(t, ok)
	assert.Equal(t, 5, r)

	h = New(5, deck.CardsFromString("2c,3c,4h,4h,2d"))
	r, ok = h.GetThreeOfAKind()
	assert.False(t, ok)
	assert.Equal(t, 0, r)
}

func TestHandAnalyzer_GetTwoPair(t *testing.T) {
	h := New(5, deck.CardsFromString("5c,5d,6h,6d,3h"))
	r, ok := h.GetTwoPair()
	assert.True(t, ok)
	assert.Equal(t, []int{6, 5}, r)

	h = New(5, deck.CardsFromString("2c,2c,3h,4h,5d"))
	r, ok = h.GetTwoPair()
	assert.False(t, ok)
	assert.Nil(t, r)
}

func TestHandAnalyzer_GetFlush(t *testing.T) {
	h := New(5, deck.CardsFromString("2c,3c,4c,5c,6c,7d,8d"))
	r, ok := h.GetFlush()
	assert.True(t, ok)
	assert.Equal(t, []int{6, 5, 4, 3, 2}, r)

	h = New(5, deck.CardsFromString("2c,3c,4c,5c,6d"))
	r, ok = h.GetFlush()
	assert.False(t, ok)
	assert.Nil(t, r)
}

func TestHandAnalyzer_GetRoyalFlush(t *testing.T) {
	h := New(5, deck.CardsFromString("10s,11s,12s,13s,14s"))
	assert.True(t, h.GetRoyalFlush())

	h = New(5, deck.CardsFromString("10s,11s,12s,8d,13s,14s,9d"))
	assert.True(t, h.GetRoyalFlush())

	h = New(3, deck.CardsFromString("14s,13s,12c,12s"))
	assert.True(t, h.GetRoyalFlush())

	h = New(3, deck.CardsFromString("14s,13s,12h,12d"))
	assert.False(t, h.GetRoyalFlush())
}

// nolint:dupl
func TestHandAnalyzer_GetStraightFlush(t *testing.T) {
	h := New(5, deck.CardsFromString("2c,3c,4c,5c,6c"))
	r, ok := h.GetStraightFlush()
	assert.True(t, ok)
	assert.Equal(t, 6, r)

	h = New(5, deck.CardsFromString("12c,2d,4h,5h,6h,14d,7h,8h"))
	r, ok = h.GetStraightFlush()
	assert.True(t, ok)
	assert.Equal(t, 8, r)

	h = New(5, deck.CardsFromString("2s,3s,4s,5s,14s"))
	r, ok = h.GetStraightFlush()
	assert.True(t, ok)
	assert.Equal(t, 5, r)

	h = New(3, deck.CardsFromString("2c,14c,3c"))
	r, ok = h.GetStraightFlush()
	assert.True(t, ok)
	assert.Equal(t, 3, r)

	h = New(3, deck.CardsFromString("2c,13c,3c"))
	r, ok = h.GetStraightFlush()
	assert.False(t, ok)
	assert.Equal(t, 0, r)
}

// nolint:dupl
func TestHandAnalyzer_GetStraight(t *testing.T) {
	h := New(5, deck.CardsFromString("2c,3d,4h,5s,6c"))
	r, ok := h.GetStraight()
	assert.True(t, ok)
	assert.Equal(t, 6, r)

	h = New(5, deck.CardsFromString("12c,2d,4h,5s,6c,14d,7d,8h"))
	r, ok = h.GetStraight()
	assert.True(t, ok)
	assert.Equal(t, 8, r)

	h = New(5, deck.CardsFromString("2c,3d,4s,5h,14s"))
	r, ok = h.GetStraight()
	assert.True(t, ok)
	assert.Equal(t, 5, r)

	h = New(3, deck.CardsFromString("2c,14s,3d"))
	r, ok = h.GetStraight()
	assert.True(t, ok)
	assert.Equal(t, 3, r)

	h = New(3, deck.CardsFromString("2c,13s,3d"))
	r, ok = h.GetStraight()
	assert.False(t, ok)
	assert.Equal(t, 0, r)
}

func TestHandAnalyzer_GetHand(t *testing.T) {
	h := New(5, deck.CardsFromString("2c,2c,2c,2c,3h"))
	assert.Equal(t, FourOfAKind, h.GetHand())
	assert.Equal(t, "Four of a kind", h.GetHand().String())

	h = New(5, deck.CardsFromString("2c,2c,2c,3c,3h"))
	assert.Equal(t, FullHouse, h.GetHand())
	assert.Equal(t, "Full house", h.GetHand().String())

	h = New(5, deck.CardsFromString("2c,2h,3c,3h,4c,5c,8c"))
	assert.Equal(t, Flush, h.GetHand())
	assert.Equal(t, "Flush", h.GetHand().String())

	h = New(5, deck.CardsFromString("2c,2c,2c,3c,4h"))
	assert.Equal(t, ThreeOfAKind, h.GetHand())
	assert.Equal(t, "Three of a kind", h.GetHand().String())

	h = New(5, deck.CardsFromString("2c,2c,3c,3c,4h"))
	assert.Equal(t, TwoPair, h.GetHand())
	assert.Equal(t, "Two pair", h.GetHand().String())

	h = New(5, deck.CardsFromString("2c,2c,3c,4c,5h"))
	assert.Equal(t, OnePair, h.GetHand())
	assert.Equal(t, "Pair", h.GetHand().String())

	h = New(5, deck.CardsFromString("2c,4c,13c,5c,8h"))
	assert.Equal(t, HighCard, h.GetHand())
	assert.Equal(t, "High card", h.GetHand().String())

	h = New(5, deck.CardsFromString("3c,4d,5h,6s,7c"))
	assert.Equal(t, Straight, h.GetHand())
	assert.Equal(t, "Straight", h.GetHand().String())

	h = New(5, deck.CardsFromString("3c,4c,5c,6c,7c"))
	assert.Equal(t, StraightFlush, h.GetHand())
	assert.Equal(t, "Straight flush", h.GetHand().String())

	h = New(5, deck.CardsFromString("14c,13c,12c,11c,10c"))
	assert.Equal(t, RoyalFlush, h.GetHand())
	assert.Equal(t, "Royal flush", h.GetHand().String())

	h = New(3, deck.CardsFromString("2c,3c,4s,8s,10s"))
	assert.Equal(t, ThreeCardPokerStraight, h.GetHand())
	assert.Equal(t, "Straight", h.GetHand().String())
}

func BenchmarkNewHandAnalyzer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := New(5, deck.CardsFromString("3s,5s,6h,7h,11c,12c,14h"))
		h.GetHand()
	}
}

func TestHandAnalyzer_GetStrength(t *testing.T) {
	type testHand struct {
		hand, name string
	}

	prevStrength := 0
	var prevTestHand testHand

	checkStrength := func(hands []testHand) {
		newHA := func(s string) *HandAnalyzer {
			cards := deck.CardsFromString(s)
			return New(len(cards), cards)
		}

		for _, h := range hands {
			hand := newHA(h.hand)
			strength := hand.GetStrength()
			if !assert.Greater(t, strength, prevStrength, fmt.Sprintf("%s is greater than %s", h.name, prevTestHand.name)) {
				log.Printf("previous hand: %s (%s)", prevTestHand.hand, newHA(prevTestHand.hand).GetHand())
				log.Printf("current hand: %s (%s)", h.hand, hand.GetHand())
			}

			prevStrength = strength
			prevTestHand = h
		}
	}

	checkStrength([]testHand{
		{"2c,3c,4c,5c,7d", "high-card (7)"},
		{"2c,3c,4c,5c,8d", "high-card (8, 5 kicker)"},
		{"2c,3c,4c,6c,8d", "high-card (8, 6 kicker)"},
		{"14c,13c,12c,11c,9d", "high-card (ace)"},
		{"2c,2d,3c,4c,5c", "one-pair (2s w/5 kicker)"},
		{"2c,2d,3c,4c,6c", "one-pair (2s w/6 kicker)"},
		{"3c,3d,2c,4c,5c", "one-pair (3s)"},
		{"14c,14d,13c,12c,11c", "one-pair (aces)"},
		{"2c,2d,3c,3d,5c", "two-pair (3s and 2s)"},
		{"2c,2d,4c,4d,3c", "two-pair (4s and 2s)"},
		{"14c,14d,13c,13d,12c", "two-pair (aces and kings)"},
		{"2c,2d,2h,3c,4c", "trips (2s)"},
		{"2c,2d,2h,3c,5c", "trips (2s w/5 kicker)"},
		{"14c,14d,14h,13c,12c", "trips (aces)"},
		{"14c,2c,3c,4c,5d", "5-high straight"},
		{"2c,3c,4c,5d,6c", "6-high straight"},
		{"10c,11c,12c,13c,14d", "ace-high straight"},
		{"2c,3c,4c,5c,8c", "8-high flush"},
		{"2c,3c,4c,6c,8c", "8-high flush (6 kicker)"},
		{"14c,13c,12c,11c,9c", "ace-high flush"},
		{"2c,3d,4h", "4-high three-card-straight"},
		{"14c,13d,12d", "ace-high three-card-straight"},
		{"2c,2d,2h", "trips (2s in three-card-poker)"},
		{"14c,14d,14h", "trips (aces in three-card-poker)"},
		{"2c,2d,2h,3c,3d", "full-house (2s over 3s"},
		{"2c,2d,3h,3c,3d", "full-house (3s over 2s"},
		{"14c,14d,14h,13c,13d", "full-house (aces over kings)"},
		{"2c,2d,2h,2s,2c", "four-of-a-kind (2s w/2 kicker)"},
		{"2c,2d,2h,2s,3c", "four-of-a-kind (2s w/3 kicker)"},
		{"14c,14d,14h,14s,13c", "four-of-a-kind (aces w/king kicker)"},
		{"14c,14d,14h,14s,14c", "four-of-a-kind (aces w/ace kicker)"},
		{"14c,2c,3c,4c,5c", "straight-flush (5-high)"},
		{"9c,10c,11c,12c,13c", "straight-flush (king-high)"},
		{"10c,11c,12c,13c,14c", "royal flush"},
	})
}

func TestHandAnalyzer_GetStrength_unknownHand(t *testing.T) {
	h := New(5, deck.CardsFromString("2c,2d,2h,2s,3c"))
	h.hand = Hand(-1)
	assert.PanicsWithValue(t, "unknown hand", func() {
		h.GetStrength()
	})
}

func TestHandAnalyzer_getThreeCardPokerThreeOfAKind(t *testing.T) {
	h := New(3, deck.CardsFromString("2c,3d,3h,3s,4c"))
	trips, ok := h.getThreeCardPokerThreeOfAKind()
	assert.True(t, ok)
	assert.Equal(t, 3, trips)

	// don't allow three card in five card game
	h = New(5, deck.CardsFromString("2c,3d,3h,3s,4c"))
	trips, ok = h.getThreeCardPokerThreeOfAKind()
	assert.False(t, ok)
	assert.Equal(t, 0, trips)

	h = New(5, deck.CardsFromString("2c,2d,3d,3h,4d"))
	trips, ok = h.getThreeCardPokerThreeOfAKind()
	assert.False(t, ok)
	assert.Equal(t, 0, trips)
}
