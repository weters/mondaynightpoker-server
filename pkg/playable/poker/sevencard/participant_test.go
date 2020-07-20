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
