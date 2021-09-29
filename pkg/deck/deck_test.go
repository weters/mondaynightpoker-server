package deck

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewDeck(t *testing.T) {
	deck := New()

	assert.Equal(t, 52, deck.CardsLeft())

	assert.Equal(t, Card{Rank: 2, Suit: Clubs}, *deck.Cards[0])

	assert.Equal(t, Card{Rank: 14, Suit: Spades}, *deck.Cards[51])

	assert.Equal(t, "79441517e1184e0e3c37383d2f7bc54996872dd8", deck.HashCode())

	deck.SetSeed(1)
	deck.Shuffle()

	assert.Equal(t, Card{Suit: Clubs, Rank: 14}, *deck.Cards[0])

	assert.Equal(t, Card{Suit: Spades, Rank: 12}, *deck.Cards[51])

	const expected = "3ba18276fa61c15ea5195929327d2bc7dda0c0c0"
	assert.Equal(t, expected, deck.HashCode())

	deck.Shuffle()

	assert.NotEqual(t, expected, deck.HashCode())
}

func TestNewFiveSuit(t *testing.T) {
	deck := NewFiveSuit()

	assert.Equal(t, 65, deck.CardsLeft())
}

func TestDeck_Draw(t *testing.T) {
	deck := New()

	if !deck.CanDraw(52) {
		t.Errorf("expected CanDraw(52) to be true")
	}

	if deck.CanDraw(53) {
		t.Errorf("expected CanDraw(53) to be false")
	}

	for i := 0; i < 52; i++ {
		card, err := deck.Draw()
		if card == nil {
			t.Error("expected card, got nil")
		}

		if err != nil {
			t.Errorf("expected err to be nil, got %v", err)
		}
	}

	if deck.CanDraw(1) {
		t.Errorf("expected CanDraw(1) to be false")
	}

	card, err := deck.Draw()
	if card != nil {
		t.Errorf("expected card to be nil, got %#v", card)
	}

	if err != ErrEndOfDeck {
		t.Errorf("expected err to be ErrEndOfDeck, got %#v", err)
	}

	deck.Shuffle()
	if !deck.CanDraw(52) {
		t.Errorf("expected Shuffle() to reshuffle the deck")
	}
}

func TestDeck_ShuffleDiscards(t *testing.T) {
	d := New()
	d.SetSeed(0)

	c1, _ := d.Draw()
	c2, _ := d.Draw()
	c3, _ := d.Draw()
	c4, _ := d.Draw()
	discards := []*Card{c1, c2, c3, c4}

	d.ShuffleDiscards(discards)
	assert.True(t, discards[0].Equal(c1))
	assert.True(t, discards[1].Equal(c2))
	assert.True(t, discards[2].Equal(c3))
	assert.True(t, discards[3].Equal(c4))

	assert.Equal(t, 4, len(d.Cards))
	assert.True(t, d.Cards[0].Equal(c4))
	assert.True(t, d.Cards[1].Equal(c2))
	assert.True(t, d.Cards[2].Equal(c1))
	assert.True(t, d.Cards[3].Equal(c3))
}

func TestDeck_RemoveCard(t *testing.T) {
	a := assert.New(t)
	d := New()
	a.True(d.RemoveCard(CardFromString("5s")))
	a.False(d.RemoveCard(CardFromString("5s")))
	a.Equal(51, len(d.Cards))

	a.True(d.RemoveCard(CardFromString("5c")))
	a.False(d.RemoveCard(CardFromString("5c")))
	a.Equal(50, len(d.Cards))
}

func TestDeck_UndoDraw(t *testing.T) {
	d := New()
	d.Cards = CardsFromString("2c,3c,4c")
	d.UndoDraw(CardFromString("14s"))

	assert.Equal(t, "14s,2c,3c,4c", CardsToString(d.Cards))
}
