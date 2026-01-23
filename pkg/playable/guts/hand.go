package guts

import "mondaynightpoker-server/pkg/deck"

// HandType represents the type of 2-card hand
type HandType int

const (
	// HighCard is a hand with two different ranks
	HighCard HandType = iota
	// Pair is a hand with two cards of the same rank
	Pair
)

// HandResult contains the analysis of a 2-card hand
type HandResult struct {
	Type     HandType
	Strength int
	HighCard int
	LowCard  int
}

// AnalyzeHand analyzes a 2-card hand and returns its type and strength
// For 2-card guts: Pair > High Card
// Strength is calculated as: Type*225 + HighRank*15 + LowRank
// This gives pairs a higher base strength than any high card hand
func AnalyzeHand(cards []*deck.Card) HandResult {
	if len(cards) != 2 {
		return HandResult{}
	}

	card1, card2 := cards[0], cards[1]
	rank1, rank2 := card1.Rank, card2.Rank

	// Determine high and low cards
	highRank, lowRank := rank1, rank2
	if rank2 > rank1 {
		highRank, lowRank = rank2, rank1
	}

	// Check for pair
	if rank1 == rank2 {
		return HandResult{
			Type:     Pair,
			Strength: int(Pair)*225 + highRank*15,
			HighCard: highRank,
			LowCard:  lowRank,
		}
	}

	// High card
	return HandResult{
		Type:     HighCard,
		Strength: int(HighCard)*225 + highRank*15 + lowRank,
		HighCard: highRank,
		LowCard:  lowRank,
	}
}

// CompareHands compares two hands and returns:
// 1 if hand1 wins, -1 if hand2 wins, 0 if tie
func CompareHands(hand1, hand2 []*deck.Card) int {
	result1 := AnalyzeHand(hand1)
	result2 := AnalyzeHand(hand2)

	if result1.Strength > result2.Strength {
		return 1
	}
	if result1.Strength < result2.Strength {
		return -1
	}
	return 0
}

// HandTypeName returns a human-readable name for the hand type
func HandTypeName(t HandType) string {
	switch t {
	case Pair:
		return "Pair"
	case HighCard:
		return "High Card"
	default:
		return "Unknown"
	}
}
