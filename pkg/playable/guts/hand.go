package guts

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
)

// HandType represents the type of hand
type HandType int

const (
	// HighCard is a hand with no matching ranks
	HighCard HandType = iota
	// Pair is a hand with two cards of the same rank
	Pair
	// ThreeCardStraight is a 3-card straight (beats flush in 3-card poker)
	ThreeCardStraight
	// Flush is a hand with all cards of the same suit (3-card only)
	Flush
	// ThreeOfAKind is a hand with three cards of the same rank
	ThreeOfAKind
)

// HandResult contains the analysis of a hand
type HandResult struct {
	Type     HandType
	Strength int
	HighCard int
	LowCard  int
}

// AnalyzeHand analyzes a hand and returns its type and strength
// For 2-card guts: Pair > High Card
// For 3-card guts: Three of a Kind > Straight > Flush > Pair > High Card
func AnalyzeHand(cards []*deck.Card) HandResult {
	if len(cards) == 2 {
		return analyze2CardHand(cards)
	} else if len(cards) == 3 {
		return analyze3CardHand(cards)
	}
	return HandResult{}
}

// analyze2CardHand analyzes a 2-card hand
// Strength is calculated as: Type*225 + HighRank*15 + LowRank
// This gives pairs a higher base strength than any high card hand
func analyze2CardHand(cards []*deck.Card) HandResult {
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

// analyze3CardHand analyzes a 3-card hand using the handanalyzer
// In 3-card poker, straights beat flushes
func analyze3CardHand(cards []*deck.Card) HandResult {
	ha := handanalyzer.New(3, cards)
	haHand := ha.GetHand()
	strength := ha.GetStrength()

	// Map handanalyzer.Hand to our HandType
	var handType HandType
	switch haHand {
	case handanalyzer.ThreeCardPokerThreeOfAKind:
		handType = ThreeOfAKind
	case handanalyzer.ThreeCardPokerStraight:
		handType = ThreeCardStraight
	case handanalyzer.Flush:
		handType = Flush
	case handanalyzer.OnePair:
		handType = Pair
	default:
		handType = HighCard
	}

	// Get high and low card
	highRank := cards[0].Rank
	lowRank := cards[0].Rank
	for _, c := range cards[1:] {
		if c.Rank > highRank {
			highRank = c.Rank
		}
		if c.Rank < lowRank {
			lowRank = c.Rank
		}
	}

	return HandResult{
		Type:     handType,
		Strength: strength,
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
	case ThreeOfAKind:
		return "Three of a Kind"
	case ThreeCardStraight:
		return "Straight"
	case Flush:
		return "Flush"
	case Pair:
		return "Pair"
	case HighCard:
		return "High Card"
	default:
		return "Unknown"
	}
}
