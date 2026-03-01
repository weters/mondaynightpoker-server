package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
)

func TestCardInfoToDeckCard(t *testing.T) {
	c := cardInfoToDeckCard(CardInfo{Rank: 14, Suit: "hearts"})
	assert.Equal(t, 14, c.Rank)
	assert.Equal(t, "hearts", string(c.Suit))

	c2 := cardInfoToDeckCard(CardInfo{Rank: 2, Suit: "clubs"})
	assert.Equal(t, 2, c2.Rank)
	assert.Equal(t, "clubs", string(c2.Suit))
}

func TestCardInfosToDeckCards(t *testing.T) {
	cards := cardInfosToDeckCards([]CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "spades"},
	})
	assert.Len(t, cards, 2)
	assert.Equal(t, 14, cards[0].Rank)
	assert.Equal(t, 13, cards[1].Rank)
}

func TestEvaluatePokerHand_RoyalFlush(t *testing.T) {
	hand := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "hearts"},
	}
	community := []CardInfo{
		{Rank: 12, Suit: "hearts"},
		{Rank: 11, Suit: "hearts"},
		{Rank: 10, Suit: "hearts"},
	}
	h, strength := evaluatePokerHand(hand, community)
	assert.Equal(t, handanalyzer.RoyalFlush, h)
	assert.Greater(t, strength, 0)
}

func TestEvaluatePokerHand_Pair(t *testing.T) {
	hand := []CardInfo{
		{Rank: 10, Suit: "hearts"},
		{Rank: 10, Suit: "spades"},
	}
	community := []CardInfo{
		{Rank: 5, Suit: "clubs"},
		{Rank: 7, Suit: "diamonds"},
		{Rank: 2, Suit: "hearts"},
	}
	h, _ := evaluatePokerHand(hand, community)
	assert.Equal(t, handanalyzer.OnePair, h)
}

func TestEvaluatePokerHand_FullHouse(t *testing.T) {
	hand := []CardInfo{
		{Rank: 8, Suit: "hearts"},
		{Rank: 8, Suit: "spades"},
	}
	community := []CardInfo{
		{Rank: 8, Suit: "clubs"},
		{Rank: 5, Suit: "diamonds"},
		{Rank: 5, Suit: "hearts"},
	}
	h, _ := evaluatePokerHand(hand, community)
	assert.Equal(t, handanalyzer.FullHouse, h)
}

func TestHandStrengthScore(t *testing.T) {
	tests := []struct {
		hand     handanalyzer.Hand
		expected float64
	}{
		{handanalyzer.RoyalFlush, 1.0},
		{handanalyzer.StraightFlush, 0.95},
		{handanalyzer.FourOfAKind, 0.90},
		{handanalyzer.FullHouse, 0.80},
		{handanalyzer.Flush, 0.70},
		{handanalyzer.Straight, 0.65},
		{handanalyzer.ThreeOfAKind, 0.55},
		{handanalyzer.TwoPair, 0.40},
		{handanalyzer.OnePair, 0.25},
		{handanalyzer.HighCard, 0.10},
	}
	for _, tt := range tests {
		t.Run(tt.hand.String(), func(t *testing.T) {
			assert.InDelta(t, tt.expected, handStrengthScore(tt.hand), 0.001)
		})
	}
}

func TestStartingHandStrength_PocketPair(t *testing.T) {
	hand := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 14, Suit: "spades"},
	}
	s := startingHandStrength(hand)
	assert.Greater(t, s, 0.7, "pocket aces should be strong")
}

func TestStartingHandStrength_Suited(t *testing.T) {
	hand := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 10, Suit: "hearts"},
	}
	s := startingHandStrength(hand)
	assert.Greater(t, s, 0.30)
}

func TestStartingHandStrength_Connected(t *testing.T) {
	hand := []CardInfo{
		{Rank: 10, Suit: "hearts"},
		{Rank: 9, Suit: "spades"},
	}
	s := startingHandStrength(hand)
	assert.Greater(t, s, 0.25)
}

func TestStartingHandStrength_Weak(t *testing.T) {
	hand := []CardInfo{
		{Rank: 7, Suit: "hearts"},
		{Rank: 2, Suit: "spades"},
	}
	s := startingHandStrength(hand)
	assert.Less(t, s, 0.20, "7-2 offsuit should be weak")
}

func TestStartingHandStrength_EmptyHand(t *testing.T) {
	s := startingHandStrength(nil)
	assert.InDelta(t, 0.10, s, 0.001)
}
