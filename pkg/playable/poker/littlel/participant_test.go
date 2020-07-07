package littlel

import (
	"github.com/bmizerany/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
	"testing"
)

func TestParticipant_GetBestHand_WithCache(t *testing.T) {
	p := &Participant{
		hand: deck.CardsFromString("2c,3c,4s,5s"),
	}

	bestHand := p.GetBestHand(deck.CardsFromString(",,"))
	assert.Equal(t, "2c,3c,4s,5s", deck.CardsToString(bestHand.hand))

	// check for proper caching
	p.bestHand.hand = deck.CardsFromString("14s")
	bestHand = p.GetBestHand(deck.CardsFromString(",,"))
	assert.Equal(t, "14s", deck.CardsToString(bestHand.hand))
}

func TestParticipant_GetBestHand(t *testing.T) {
	p := &Participant{
		hand: deck.CardsFromString("2c,3d,4h,4s"),
	}

	bestHand := p.GetBestHand(deck.CardsFromString(",,"))
	assert.Equal(t, "2c,3d,4h,4s", deck.CardsToString(bestHand.hand))
	assert.Equal(t, handanalyzer.ThreeCardPokerStraight.String(), bestHand.analyzer.GetHand().String())
	rank, _ := bestHand.analyzer.GetStraight()
	assert.Equal(t, 4, rank)

	bestHand = p.GetBestHand(deck.CardsFromString("5d,,"))
	assert.Equal(t, "2c,3d,4h,4s,5d", deck.CardsToString(bestHand.hand))
	assert.Equal(t, handanalyzer.Straight.String(), bestHand.analyzer.GetHand().String())
	rank, _ = bestHand.analyzer.GetStraight()
	assert.Equal(t, 5, rank)

	bestHand = p.GetBestHand(deck.CardsFromString("5d,3c,"))
	assert.Equal(t, "2c,3d,4h,4s,5d", deck.CardsToString(bestHand.hand))
	assert.Equal(t, handanalyzer.Straight.String(), bestHand.analyzer.GetHand().String())
	rank, _ = bestHand.analyzer.GetStraight()
	assert.Equal(t, 5, rank)

	bestHand = p.GetBestHand(deck.CardsFromString("5d,3c,4c"))
	assert.Equal(t, "2c,3d,4h,4s,3c,4c", deck.CardsToString(bestHand.hand))
	assert.Equal(t, handanalyzer.StraightFlush.String(), bestHand.analyzer.GetHand().String())
	rank, _ = bestHand.analyzer.GetStraightFlush()
	assert.Equal(t, 4, rank)

	//bestHand := p.GetBestHand(deck.CardsFromString(",,"))
	//assert.Equal(t, "2c,3c,4s,5s", deck.CardsToString(bestHand.hand))
}
