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

	card = Card{
		Rank: 2,
		Suit: Stars,
	}

	assert.Equal(t, "2☆", card.String())

	card = Card{
		Rank: 3,
		Suit: "Bad",
	}

	assert.PanicsWithValue(t, "unknown suit", func() {
		_ = card.String()
	})
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
	assert.False(t, c.IsWild)

	c = CardFromString("!3d")
	assert.Equal(t, 3, c.Rank)
	assert.Equal(t, Diamonds, c.Suit)
	assert.True(t, c.IsWild)

	c = CardFromString("4h")
	assert.Equal(t, 4, c.Rank)
	assert.Equal(t, Hearts, c.Suit)

	c = CardFromString("14S")
	assert.Equal(t, 14, c.Rank)
	assert.Equal(t, Spades, c.Suit)

	c = CardFromString("14T")
	assert.Equal(t, 14, c.Rank)
	assert.Equal(t, Stars, c.Suit)

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

func TestCardToString(t *testing.T) {
	assert.Equal(t, "14c", CardToString(&Card{
		Rank: Ace,
		Suit: Clubs,
	}))

	assert.Equal(t, "!14c", CardToString(&Card{
		Rank:   Ace,
		Suit:   Clubs,
		IsWild: true,
	}))

	assert.Equal(t, "3t", CardToString(&Card{
		Rank: 3,
		Suit: Stars,
	}))

	assert.Equal(t, "", CardToString(nil))
}

func TestCard_Wild(t *testing.T) {
	a := assert.New(t)

	c := CardFromString("!13c")
	a.Equal(13, c.GetWildRank())
	a.Equal(Clubs, c.GetWildSuit())

	a.NoError(c.SetWildRank(5))
	a.NoError(c.SetWildSuit(Diamonds))

	a.Equal(5, c.GetWildRank())
	a.Equal(Diamonds, c.GetWildSuit())

	c = CardFromString("13c")
	a.Equal(ErrNotWild, c.SetWildRank(5))
	a.Equal(ErrNotWild, c.SetWildSuit(Diamonds))
	a.Equal(13, c.GetWildRank())
	a.Equal(Clubs, c.GetWildSuit())
}

func TestCard_Clone(t *testing.T) {
	c := CardFromString("14s")
	cp := c.Clone()
	c.Rank = 13
	c.Suit = Clubs
	assert.NotEqual(t, cp.Suit, c.Suit)
	assert.NotEqual(t, cp.Rank, c.Rank)
}

func TestCard_bitField(t *testing.T) {
	const (
		bitA int = 1 << iota
		bitB
		bitC
	)

	c := CardFromString("14s")

	a := assert.New(t)
	a.False(c.IsBitSet(bitA))
	a.False(c.IsBitSet(bitB))
	a.False(c.IsBitSet(bitC))

	c.SetBit(bitB)
	a.False(c.IsBitSet(bitA))
	a.True(c.IsBitSet(bitB))
	a.False(c.IsBitSet(bitC))

	c.SetBit(bitC)
	c.UnsetBit(bitB)
	a.False(c.IsBitSet(bitA))
	a.False(c.IsBitSet(bitB))
	a.True(c.IsBitSet(bitC))
}

func TestCard_UnsetAllBits(t *testing.T) {
	c := CardFromString("14s")
	c.BitField = 15
	c.UnsetAllBits()

	assert.Equal(t, 0, c.BitField)
}
