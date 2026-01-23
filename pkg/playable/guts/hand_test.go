package guts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
)

func TestAnalyzeHand_Pair(t *testing.T) {
	tests := []struct {
		name     string
		cards    string
		wantType HandType
		highCard int
	}{
		{"Pair of Aces", "14c,14d", Pair, 14},
		{"Pair of Kings", "13c,13d", Pair, 13},
		{"Pair of Twos", "2c,2d", Pair, 2},
		{"Pair of Tens", "10c,10d", Pair, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cards := deck.CardsFromString(tt.cards)
			result := AnalyzeHand(cards)
			assert.Equal(t, tt.wantType, result.Type)
			assert.Equal(t, tt.highCard, result.HighCard)
		})
	}
}

func TestAnalyzeHand_HighCard(t *testing.T) {
	tests := []struct {
		name     string
		cards    string
		wantType HandType
		highCard int
		lowCard  int
	}{
		{"Ace-King", "14c,13d", HighCard, 14, 13},
		{"Ace-Two", "14c,2d", HighCard, 14, 2},
		{"King-Queen", "13c,12d", HighCard, 13, 12},
		{"Ten-Five", "10c,5d", HighCard, 10, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cards := deck.CardsFromString(tt.cards)
			result := AnalyzeHand(cards)
			assert.Equal(t, tt.wantType, result.Type)
			assert.Equal(t, tt.highCard, result.HighCard)
			assert.Equal(t, tt.lowCard, result.LowCard)
		})
	}
}

func TestCompareHands_PairBeatHighCard(t *testing.T) {
	// Pair of 2s beats Ace-King high
	pairOf2s := deck.CardsFromString("2c,2d")
	aceKing := deck.CardsFromString("14c,13d")

	result := CompareHands(pairOf2s, aceKing)
	assert.Equal(t, 1, result, "Pair of 2s should beat Ace-King")

	result = CompareHands(aceKing, pairOf2s)
	assert.Equal(t, -1, result, "Ace-King should lose to Pair of 2s")
}

func TestCompareHands_HigherPairWins(t *testing.T) {
	pairOfAces := deck.CardsFromString("14c,14d")
	pairOfKings := deck.CardsFromString("13c,13d")

	result := CompareHands(pairOfAces, pairOfKings)
	assert.Equal(t, 1, result, "Pair of Aces should beat Pair of Kings")

	result = CompareHands(pairOfKings, pairOfAces)
	assert.Equal(t, -1, result, "Pair of Kings should lose to Pair of Aces")
}

func TestCompareHands_PairTie(t *testing.T) {
	pairOfAces1 := deck.CardsFromString("14c,14d")
	pairOfAces2 := deck.CardsFromString("14h,14s")

	result := CompareHands(pairOfAces1, pairOfAces2)
	assert.Equal(t, 0, result, "Same rank pairs should tie")
}

func TestCompareHands_HighCardComparison(t *testing.T) {
	// Ace-King beats Ace-Queen
	aceKing := deck.CardsFromString("14c,13d")
	aceQueen := deck.CardsFromString("14c,12d")

	result := CompareHands(aceKing, aceQueen)
	assert.Equal(t, 1, result, "Ace-King should beat Ace-Queen")

	// King-Queen beats King-Jack
	kingQueen := deck.CardsFromString("13c,12d")
	kingJack := deck.CardsFromString("13c,11d")

	result = CompareHands(kingQueen, kingJack)
	assert.Equal(t, 1, result, "King-Queen should beat King-Jack")
}

func TestCompareHands_HighCardTie(t *testing.T) {
	aceKing1 := deck.CardsFromString("14c,13d")
	aceKing2 := deck.CardsFromString("14h,13s")

	result := CompareHands(aceKing1, aceKing2)
	assert.Equal(t, 0, result, "Same high cards should tie")
}

func TestHandTypeName(t *testing.T) {
	assert.Equal(t, "Pair", HandTypeName(Pair))
	assert.Equal(t, "High Card", HandTypeName(HighCard))
	assert.Equal(t, "Unknown", HandTypeName(HandType(99)))
}

func TestAnalyzeHand_EmptyHand(t *testing.T) {
	result := AnalyzeHand([]*deck.Card{})
	assert.Equal(t, HandResult{}, result)
}

func TestAnalyzeHand_SingleCard(t *testing.T) {
	cards := deck.CardsFromString("14c")
	result := AnalyzeHand(cards)
	assert.Equal(t, HandResult{}, result)
}

func TestStrengthCalculation(t *testing.T) {
	// Verify that any pair beats any high card
	lowestPair := AnalyzeHand(deck.CardsFromString("2c,2d"))
	highestHighCard := AnalyzeHand(deck.CardsFromString("14c,13d"))

	assert.Greater(t, lowestPair.Strength, highestHighCard.Strength,
		"Lowest pair (2s) should have higher strength than highest high card (AK)")
}
