package sevencard

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestLowCardWild_Name(t *testing.T) {
	opts := DefaultOptions()
	opts.Variant = &LowCardWild{}
	game, _ := NewGame("", []int64{1, 2}, opts)
	assert.Equal(t, "Low Card Wild", game.Name())
}

func TestLowCardWild_ParticipantReceivedCard(t *testing.T) {
	a := assert.New(t)
	lw := &LowCardWild{}

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}

		return card
	}

	p := newParticipant(1, 25)

	assertCard := func(n int, expects string, isPrivateWild bool) {
		card := p.hand[n]
		a.Equal(expects, deck.CardToString(card))
		a.Equal(isPrivateWild, card.IsBitSet(privateWild))
	}

	p.hand.AddCard(c("8c", false))
	lw.ParticipantReceivedCard(nil, p, nil)
	assertCard(0, "!8c", false)

	p.hand.AddCard(c("3c", false))
	lw.ParticipantReceivedCard(nil, p, nil)
	assertCard(0, "8c", false)
	assertCard(1, "!3c", false)

	p.hand.AddCard(c("2c", true))
	lw.ParticipantReceivedCard(nil, p, nil)
	assertCard(0, "8c", false)
	assertCard(1, "!3c", false)
	assertCard(2, "2c", false)

	p.hand.AddCard(c("3d", true))
	lw.ParticipantReceivedCard(nil, p, nil)
	assertCard(0, "8c", false)
	assertCard(1, "!3c", false)
	assertCard(2, "2c", false)
	assertCard(3, "!3d", true)

	p.hand.AddCard(c("2d", false))
	lw.ParticipantReceivedCard(nil, p, nil)
	assertCard(0, "8c", false)
	assertCard(1, "3c", false)
	assertCard(2, "!2c", true)
	assertCard(3, "3d", false)
	assertCard(4, "!2d", false)
}
