package handanalyzer

import (
	"mondaynightpoker-server/pkg/deck"
)

// WildMode determines how a wild card can be used
type WildMode int

const (
	// RankMode means the wild keeps its original suit but can represent any rank
	RankMode WildMode = iota
	// SuitMode means the wild keeps its original rank but can represent any suit
	SuitMode
)

// WildAssignment pairs a wild card with its mode
type WildAssignment struct {
	Card *deck.Card
	Mode WildMode
}

// generateWildCombinations generates all 2^n mode combinations for n wild cards
// For example, 2 wilds generates 4 combinations:
// [{Rank, Rank}, {Rank, Suit}, {Suit, Rank}, {Suit, Suit}]
func generateWildCombinations(wilds deck.Hand) [][]WildAssignment {
	n := len(wilds)
	if n == 0 {
		return [][]WildAssignment{{}}
	}

	// 2^n combinations
	numCombinations := 1 << n
	result := make([][]WildAssignment, numCombinations)

	for i := 0; i < numCombinations; i++ {
		assignment := make([]WildAssignment, n)
		for j, wild := range wilds {
			mode := RankMode
			if (i>>j)&1 == 1 {
				mode = SuitMode
			}
			assignment[j] = WildAssignment{
				Card: wild,
				Mode: mode,
			}
		}
		result[i] = assignment
	}

	return result
}

// countWildsByMode counts how many wilds are in each mode
func countWildsByMode(assignments []WildAssignment) (rankMode, suitMode int) {
	for _, a := range assignments {
		if a.Mode == RankMode {
			rankMode++
		} else {
			suitMode++
		}
	}
	return
}

// getSuitModeWilds returns only the wilds in SuitMode
func getSuitModeWilds(assignments []WildAssignment) []WildAssignment {
	result := make([]WildAssignment, 0, len(assignments))
	for _, a := range assignments {
		if a.Mode == SuitMode {
			result = append(result, a)
		}
	}
	return result
}

// getRankModeWildsForSuit returns rank-mode wilds that match the given suit
func getRankModeWildsForSuit(assignments []WildAssignment, suit deck.Suit) []WildAssignment {
	result := make([]WildAssignment, 0)
	for _, a := range assignments {
		if a.Mode == RankMode && a.Card.Suit == suit {
			result = append(result, a)
		}
	}
	return result
}

// getSuitModeWildsForRank returns suit-mode wilds that match the given rank
func getSuitModeWildsForRank(assignments []WildAssignment, rank int) []WildAssignment {
	result := make([]WildAssignment, 0)
	for _, a := range assignments {
		if a.Mode == SuitMode && a.Card.Rank == rank {
			result = append(result, a)
		}
	}
	return result
}
