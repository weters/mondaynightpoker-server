package deck

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_constants(t *testing.T) {
	assert.Equal(t, 11, Jack)
	assert.Equal(t, 12, Queen)
	assert.Equal(t, 13, King)
	assert.Equal(t, 14, Ace)
	assert.Equal(t, 1, LowAce)
	assert.Equal(t, 14, HighAce)
}

func TestCard_String(t *testing.T) {
	card := Card{
		Rank: 2,
		Suit: Hearts,
	}

	assert.Equal(t, "2♡", card.String())

	card = Card{
		Rank: 11,
		Suit: Clubs,
	}

	assert.Equal(t, "J♣", card.String())

	card = Card{
		Rank: 12,
		Suit: Diamonds,
	}

	assert.Equal(t, "Q♢", card.String())

	card = Card{
		Rank: 13,
		Suit: Spades,
	}

	assert.Equal(t, "K♠", card.String())

	card = Card{
		Rank: 14,
		Suit: Spades,
	}

	assert.Equal(t, "A♠", card.String())
}

func TestCard_AceLowRank(t *testing.T) {
	card := &Card{Rank: 2}
	assert.Equal(t, 2, card.AceLowRank())

	card.Rank = 13
	assert.Equal(t, King, card.AceLowRank())

	card.Rank = 14
	assert.Equal(t, 1, card.AceLowRank())
}

func TestCardFromString(t *testing.T) {
	c := CardFromString("2c")
	assert.Equal(t, 2, c.Rank)
	assert.Equal(t, Clubs, c.Suit)

	c = CardFromString("3d")
	assert.Equal(t, 3, c.Rank)
	assert.Equal(t, Diamonds, c.Suit)

	c = CardFromString("4h")
	assert.Equal(t, 4, c.Rank)
	assert.Equal(t, Hearts, c.Suit)

	c = CardFromString("14S")
	assert.Equal(t, 14, c.Rank)
	assert.Equal(t, Spades, c.Suit)

	assert.PanicsWithValue(t, "could not parse card: 15d", func() {
		CardFromString("15d")
	})
}

func TestCardsFromString(t *testing.T) {
	cards := CardsFromString("2c,3s")
	assert.Equal(t, "2c,3s", CardsToString(cards))

	cards = CardsFromString("")
	assert.Equal(t, []*Card{}, cards)

	cards = CardsFromString("2c,,3c")
	assert.Equal(t, "2c,,3c", CardsToString(cards))

	assert.PanicsWithValue(t, "could not parse card: 4x", func() {
		CardsFromString("2c,3s,4x")
	})
}
