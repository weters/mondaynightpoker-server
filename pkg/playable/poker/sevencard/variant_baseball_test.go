package sevencard

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestBaseball_ParticipantReceivedCard(t *testing.T) {
	opts := DefaultOptions()
	b := &Baseball{}
	opts.Variant = b
	game, _ := NewGame("", []int64{1, 2, 3, 4, 5, 6}, opts)

	a := assert.New(t)
	p := game.idToParticipant[1]

	c := deck.CardFromString("3c")
	b.ParticipantReceivedCard(game, p, c)
	a.Equal("!3c", deck.CardToString(c))

	c = deck.CardFromString("9d")
	b.ParticipantReceivedCard(game, p, c)
	a.Equal("!9d", deck.CardToString(c))

	c = deck.CardFromString("8d")
	b.ParticipantReceivedCard(game, p, c)
	a.Equal("8d", deck.CardToString(c))

	c = deck.CardFromString("4d")
	b.ParticipantReceivedCard(game, p, c)
	a.Equal("4d", deck.CardToString(c))
	a.Equal(0, b.extraCards)

	c.SetBit(faceUp)
	b.ParticipantReceivedCard(game, p, c)
	a.Equal("4d", deck.CardToString(c))
	a.Equal(1, b.extraCards)
	a.Equal(1, len(p.hand))

	b.extraCards = 3
	b.ParticipantReceivedCard(game, p, c)
	a.Equal("4d", deck.CardToString(c))
	a.Equal(4, b.extraCards)
	a.Equal(2, len(p.hand))

	game, _ = NewGame("", []int64{1, 2, 3, 4, 5, 6, 7}, opts)
	p = game.idToParticipant[1]
	b.extraCards = 3
	b.ParticipantReceivedCard(game, p, c)
	a.Equal("4d", deck.CardToString(c))
	a.Equal(3, b.extraCards)
	a.Equal(0, len(p.hand))
}

func TestBaseball_Start(t *testing.T) {
	b := &Baseball{extraCards: 5}
	b.Start()
	assert.Equal(t, 0, b.extraCards)
}

func TestBaseball_Name(t *testing.T) {
	b := Baseball{}
	assert.Equal(t, "Baseball", b.Name())
}
