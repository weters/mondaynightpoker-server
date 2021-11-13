package littlel

import (
	"github.com/stretchr/testify/assert"
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
}

func TestParticipant_GetBestHand_CacheAfterTrade(t *testing.T) {
	game := mustNewGame(DefaultOptions(), 50, 50)
	assert.NoError(t, game.DealCards())
	p := func(id int64) *Participant {
		return game.idToParticipant[id]
	}

	game.deck.Cards = deck.CardsFromString("3d,8d")

	p(1).hand = deck.CardsFromString("2d,3h,4d,9h")
	assert.Equal(t, handanalyzer.Straight.String(), p(1).GetBestHand(deck.CardsFromString(",,")).analyzer.GetHand().String())

	assert.NoError(t, game.tradeCardsForParticipant(p(1), deck.CardsFromString("3h,9h")))
	assert.NoError(t, game.tradeCardsForParticipant(p(2), []*deck.Card{}))
	assert.NoError(t, game.NextRound())

	assert.Equal(t, handanalyzer.StraightFlush.String(), p(1).GetBestHand(deck.CardsFromString(",,")).analyzer.GetHand().String())
}
