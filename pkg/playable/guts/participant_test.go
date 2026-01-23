package guts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
)

func TestNewParticipant(t *testing.T) {
	p := NewParticipant(123)

	assert.Equal(t, int64(123), p.PlayerID)
	assert.Equal(t, 0, p.balance)
	assert.Empty(t, p.hand)
}

func TestParticipant_AddCard(t *testing.T) {
	p := NewParticipant(1)

	card1 := deck.CardFromString("14c")
	card2 := deck.CardFromString("13d")

	p.AddCard(card1)
	assert.Len(t, p.hand, 1)

	p.AddCard(card2)
	assert.Len(t, p.hand, 2)
}

func TestParticipant_Hand(t *testing.T) {
	p := NewParticipant(1)

	card1 := deck.CardFromString("14c")
	card2 := deck.CardFromString("13d")
	p.AddCard(card1)
	p.AddCard(card2)

	hand := p.Hand()
	assert.Len(t, hand, 2)

	// Verify it's a copy
	hand[0] = deck.CardFromString("2c")
	assert.Equal(t, card1, p.hand[0])
}

func TestParticipant_ClearHand(t *testing.T) {
	p := NewParticipant(1)

	p.AddCard(deck.CardFromString("14c"))
	p.AddCard(deck.CardFromString("13d"))
	assert.Len(t, p.hand, 2)

	p.ClearHand()
	assert.Empty(t, p.hand)
}
