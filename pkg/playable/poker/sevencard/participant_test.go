package sevencard

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func Test_participant_getHandAnalyzer(t *testing.T) {
	p := &participant{
		hand: deck.CardsFromString("2c,3c,4c,5c,6d"),
	}

	ha := p.getHandAnalyzer()
	assert.Equal(t, "Straight", ha.GetHand().String())
	assert.Equal(t, ha, p.getHandAnalyzer(), "uses cache")

	p.hand.AddCard(deck.CardFromString("6c"))
	assert.NotEqual(t, ha, p.getHandAnalyzer(), "cache is busted")
	assert.Equal(t, "Straight flush", p.getHandAnalyzer().GetHand().String())
}

func Test_participant_getHandAnalyzer_filtersDiscardedCards(t *testing.T) {
	a := assert.New(t)

	// Create a hand where the 6d would make it a straight
	// But if 6d is discarded, it's just a high card
	p := &participant{
		hand: deck.CardsFromString("2c,3c,4c,5c,6d"),
	}

	// Without discard, should be a straight
	ha := p.getHandAnalyzer()
	a.Equal("Straight", ha.GetHand().String())

	// Mark the 6d as discarded
	p.hand[4].SetBit(wasDiscarded)

	// Should no longer be a straight (only 4 cards now)
	ha = p.getHandAnalyzer()
	a.Equal("High card", ha.GetHand().String(), "discarded card should not count toward hand")
}

func Test_participant_getHandAnalyzer_cacheUpdatesWhenCardDiscarded(t *testing.T) {
	a := assert.New(t)

	p := &participant{
		hand: deck.CardsFromString("2c,3c,4c,5c,6d"),
	}

	ha1 := p.getHandAnalyzer()
	a.Equal("Straight", ha1.GetHand().String())

	// Mark a card as discarded - cache should be invalidated
	p.hand[4].SetBit(wasDiscarded)

	ha2 := p.getHandAnalyzer()
	a.NotEqual(ha1, ha2, "cache should be invalidated when card is discarded")
}
