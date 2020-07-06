package handanalyzer

import (
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
	h := New(5, deck.CardsFromString("14s,13c,12d,11s,9c"))
	assert.Equal(t, 755499, h.GetStrength())
	h = New(5, deck.CardsFromString("14s,13c,12d,11s,9c,2c"))
	highCard := h.GetStrength()
	assert.Equal(t, 755499, highCard)

	h = New(5, deck.CardsFromString("2c,2c,12c,13h,14s"))
	onePair := h.GetStrength()
	assert.Equal(t, 910980, onePair)
	assert.Greater(t, onePair, highCard)

	h = New(5, deck.CardsFromString("3c,3c,2c,4c,5h"))
	onePairHigher := h.GetStrength()
	assert.Equal(t, 929055, onePairHigher)
	assert.Greater(t, onePairHigher, onePair)

	h = New(5, deck.CardsFromString("14s,14c,13h,12d,11c"))
	onePairHighest := h.GetStrength()

	h = New(5, deck.CardsFromString("2c,2h,3c,3h,4s"))
	twoPairLowest := h.GetStrength()
	assert.Greater(t, twoPairLowest, onePairHighest)

	h = New(5, deck.CardsFromString("14s,14c,13h,13c,12d"))
	twoPairHighest := h.GetStrength()
	h = New(5, deck.CardsFromString("2c,2h,2d,3c,4d"))
	tripsLowest := h.GetStrength()
	assert.Greater(t, tripsLowest, twoPairHighest)

	h = New(5, deck.CardsFromString("14s,14c,14d,13c,12s"))
	tripsHighest := h.GetStrength()
	h = New(5, deck.CardsFromString("14s,2c,3d,4h,5s"))
	straightLowest := h.GetStrength()
	assert.Greater(t, straightLowest, tripsHighest)

	h = New(5, deck.CardsFromString("2c,3d,4h,5s,6c"))
	straightSecondLowest := h.GetStrength()
	assert.Greater(t, straightSecondLowest, straightLowest)

	h = New(5, deck.CardsFromString("10c,11d,12h,13s,14c"))
	straightHighest := h.GetStrength()
	assert.Greater(t, straightHighest, straightLowest)
	h = New(5, deck.CardsFromString("2c,3c,4c,5c,7c"))
	flushLowest := h.GetStrength()
	assert.Greater(t, flushLowest, straightHighest)

	h = New(5, deck.CardsFromString("14c,13d,12h,11s,10c"))
	flushHighest := h.GetStrength()
	h = New(3, deck.CardsFromString("14c,2d,3s"))
	threeCardStraightLowest := h.GetStrength()
	assert.Greater(t, threeCardStraightLowest, flushHighest)

	h = New(5, deck.CardsFromString("14c,13d,12s"))
	threeCardStraightHighest := h.GetStrength()
	h = New(5, deck.CardsFromString("2c,2d,2h,3c,3d"))
	fullHouseLowest := h.GetStrength()
	assert.Greater(t, fullHouseLowest, threeCardStraightHighest)

	h = New(5, deck.CardsFromString("14c,14d,14h,13c,13d"))
	fullHouseHighest := h.GetStrength()
	h = New(5, deck.CardsFromString("2c,2d,2h,2s,3c"))
	fourOfAKindLowest := h.GetStrength()
	assert.Greater(t, fourOfAKindLowest, fullHouseHighest)

	h = New(5, deck.CardsFromString("2c,2d,2h,2s,5c"))
	fourOfAKindSecondLowest := h.GetStrength()
	assert.Greater(t, fourOfAKindSecondLowest, fourOfAKindLowest)

	h = New(5, deck.CardsFromString("3c,3d,3h,3s,3c"))
	fourOfAKindThirdLowest := h.GetStrength()
	assert.Greater(t, fourOfAKindThirdLowest, fourOfAKindSecondLowest)

	h = New(5, deck.CardsFromString("3c,3d,3h,3s,4c"))
	fourOfAKindFourthLowest := h.GetStrength()
	assert.Greater(t, fourOfAKindFourthLowest, fourOfAKindThirdLowest)

	h = New(5, deck.CardsFromString("14c,2c,3c,4c,5c"))
	straightFlushLowest := h.GetStrength()
	assert.Greater(t, straightFlushLowest, fullHouseHighest)

	h = New(5, deck.CardsFromString("13c,12c,11c,10c,9c"))
	straightFlushHighest := h.GetStrength()
	assert.Greater(t, straightFlushHighest, straightFlushLowest)

	h = New(5, deck.CardsFromString("14c,13c,12c,11c,10c"))
	royalFlush := h.GetStrength()
	assert.Greater(t, royalFlush, straightFlushHighest)
}
