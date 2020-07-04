package deck

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
