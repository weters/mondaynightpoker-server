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

// 3-Card Guts Tests

func TestAnalyze3CardHand_ThreeOfAKind(t *testing.T) {
	cards := deck.CardsFromString("14c,14d,14h")
	result := AnalyzeHand(cards)
	assert.Equal(t, ThreeOfAKind, result.Type)
	assert.Equal(t, 14, result.HighCard)
}

func TestAnalyze3CardHand_Straight(t *testing.T) {
	cards := deck.CardsFromString("12c,13d,14h")
	result := AnalyzeHand(cards)
	assert.Equal(t, ThreeCardStraight, result.Type)
	assert.Equal(t, 14, result.HighCard)
}

func TestAnalyze3CardHand_Flush(t *testing.T) {
	cards := deck.CardsFromString("14c,10c,5c")
	result := AnalyzeHand(cards)
	assert.Equal(t, Flush, result.Type)
	assert.Equal(t, 14, result.HighCard)
}

func TestAnalyze3CardHand_Pair(t *testing.T) {
	cards := deck.CardsFromString("14c,14d,10h")
	result := AnalyzeHand(cards)
	assert.Equal(t, Pair, result.Type)
	assert.Equal(t, 14, result.HighCard)
}

func TestAnalyze3CardHand_HighCard(t *testing.T) {
	cards := deck.CardsFromString("14c,10d,5h")
	result := AnalyzeHand(cards)
	assert.Equal(t, HighCard, result.Type)
	assert.Equal(t, 14, result.HighCard)
	assert.Equal(t, 5, result.LowCard)
}

func TestCompare3CardHands_StraightBeatsFlush(t *testing.T) {
	// In 3-card poker, straights beat flushes
	straight := deck.CardsFromString("12c,13d,14h")
	flush := deck.CardsFromString("14c,10c,5c")

	result := CompareHands(straight, flush)
	assert.Equal(t, 1, result, "Straight should beat Flush in 3-card poker")

	result = CompareHands(flush, straight)
	assert.Equal(t, -1, result, "Flush should lose to Straight in 3-card poker")
}

func TestCompare3CardHands_ThreeOfAKindBeatsStraight(t *testing.T) {
	threeOfAKind := deck.CardsFromString("2c,2d,2h")
	straight := deck.CardsFromString("12c,13d,14h")

	result := CompareHands(threeOfAKind, straight)
	assert.Equal(t, 1, result, "Three of a Kind should beat Straight")

	result = CompareHands(straight, threeOfAKind)
	assert.Equal(t, -1, result, "Straight should lose to Three of a Kind")
}

func TestCompare3CardHands_ThreeOfAKindBeatsFlush(t *testing.T) {
	threeOfAKind := deck.CardsFromString("2c,2d,2h")
	// Use a non-straight flush (A-10-5)
	flush := deck.CardsFromString("14c,10c,5c")

	result := CompareHands(threeOfAKind, flush)
	assert.Equal(t, 1, result, "Three of a Kind should beat Flush")
}

func TestCompare3CardHands_FlushBeatsPair(t *testing.T) {
	flush := deck.CardsFromString("14c,10c,5c")
	pair := deck.CardsFromString("14d,14h,10s")

	result := CompareHands(flush, pair)
	assert.Equal(t, 1, result, "Flush should beat Pair")

	result = CompareHands(pair, flush)
	assert.Equal(t, -1, result, "Pair should lose to Flush")
}

func TestCompare3CardHands_PairBeatsHighCard(t *testing.T) {
	pair := deck.CardsFromString("2c,2d,5h")
	// Use a non-straight high card (A-10-5)
	highCard := deck.CardsFromString("14c,10d,5h")

	result := CompareHands(pair, highCard)
	assert.Equal(t, 1, result, "Pair should beat High Card")

	result = CompareHands(highCard, pair)
	assert.Equal(t, -1, result, "High Card should lose to Pair")
}

func TestCompare3CardHands_HigherStraightWins(t *testing.T) {
	higherStraight := deck.CardsFromString("12c,13d,14h") // A-high straight
	lowerStraight := deck.CardsFromString("11c,12d,13h")  // K-high straight

	result := CompareHands(higherStraight, lowerStraight)
	assert.Equal(t, 1, result, "A-high straight should beat K-high straight")
}

func TestCompare3CardHands_WheelStraight(t *testing.T) {
	// A-2-3 is the lowest straight (wheel)
	wheel := deck.CardsFromString("14c,2d,3h")
	result := AnalyzeHand(wheel)
	assert.Equal(t, ThreeCardStraight, result.Type, "A-2-3 should be a straight")

	// Regular straight should beat wheel
	regularStraight := deck.CardsFromString("4c,5d,6h")
	compareResult := CompareHands(regularStraight, wheel)
	assert.Equal(t, 1, compareResult, "4-5-6 straight should beat A-2-3 wheel")
}

func TestHandTypeName_3Card(t *testing.T) {
	assert.Equal(t, "Three of a Kind", HandTypeName(ThreeOfAKind))
	assert.Equal(t, "Straight", HandTypeName(ThreeCardStraight))
	assert.Equal(t, "Flush", HandTypeName(Flush))
	assert.Equal(t, "Pair", HandTypeName(Pair))
	assert.Equal(t, "High Card", HandTypeName(HighCard))
}
