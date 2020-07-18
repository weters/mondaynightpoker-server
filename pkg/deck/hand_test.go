package deck

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestHand_Sort(t *testing.T) {
	hand := Hand(CardsFromString("14c,14s,14h,14d,2d,2h,2s,2c"))
	assert.Equal(t, "14c,14s,14h,14d,2d,2h,2s,2c", CardsToString(hand))
	sort.Sort(hand)
	assert.Equal(t, "2c,14c,2d,14d,2h,14h,2s,14s", CardsToString(hand))
}

func TestHand_HasCard(t *testing.T) {
	hand := Hand(CardsFromString("2c,3c,4d"))
	assert.True(t, hand.HasCard(CardFromString("3c")))
	assert.False(t, hand.HasCard(CardFromString("3s")))
}

func TestHand_Discard(t *testing.T) {
	hand := Hand(CardsFromString("2c,3c,3c,4d"))
	assert.Equal(t, 2, hand.Discard(CardFromString("3c")))
	assert.Equal(t, "2c,4d", CardsToString(hand))

	hand = Hand(CardsFromString("2c,3c,3c,4d"))
	assert.Equal(t, 1, hand.Discard(CardFromString("3c"), 1))
	assert.Equal(t, "2c,3c,4d", CardsToString(hand))
}

func TestHand_AddCard(t *testing.T) {
	h := make(Hand, 0)
	h.AddCard(CardFromString("14s"))
	h.AddCard(CardFromString("3c"))
	assert.Equal(t, "14s,3c", CardsToString(h))
}

func TestHand_LastCard(t *testing.T) {
	h := Hand(CardsFromString("2c,3d,4h"))
	assert.Equal(t, "4h", CardToString(h.LastCard()))

	h = make(Hand, 0)
	assert.Nil(t, h.LastCard())
}

func TestHand_FirstCard(t *testing.T) {
	h := Hand(CardsFromString("2c,3d,4h"))
	assert.Equal(t, "2c", CardToString(h.FirstCard()))

	h = make(Hand, 0)
	assert.Nil(t, h.FirstCard())
}

func TestHand_String(t *testing.T) {
	h := Hand(CardsFromString("2c,3d,4h"))
	assert.Equal(t, "2c,3d,4h", h.String())
}
