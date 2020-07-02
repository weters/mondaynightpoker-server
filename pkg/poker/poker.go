package poker

import (
	"fmt"
	"math"
	"mondaynightpoker-server/pkg/deck"
	"sort"
)

// Hand is a poker hand, i.e., royal flush
type Hand int

// Constants for hand
const (
	HighCard Hand = iota
	OnePair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	ThreeCardPokerStraight // in three-card poker, straight beats flush
	FullHouse
	FourOfAKind
	StraightFlush
	RoyalFlush
)

// String returns the string representation of a hand
func (h Hand) String() string {
	switch h {
	case HighCard:
		return "High card"
	case OnePair:
		return "Pair"
	case TwoPair:
		return "Two pair"
	case ThreeOfAKind:
		return "Three of a kind"
	case Straight:
		return "Straight"
	case Flush:
		return "Flush"
	case ThreeCardPokerStraight:
		return "Straight"
	case FullHouse:
		return "Full house"
	case FourOfAKind:
		return "Four of a kind"
	case StraightFlush:
		return "Straight flush"
	case RoyalFlush:
		return "Royal flush"
	default:
		panic(fmt.Sprintf("unknown hand: %d", h))
	}
}

// HandAnalyzer can analyze a hand
type HandAnalyzer struct {
	size  int
	cards []*deck.Card
	flush []int
	quads []int
	trips []int
	pairs []int

	hand Hand
}

type sortByRank []*deck.Card

func (s sortByRank) Len() int {
	return len(s)
}

func (s sortByRank) Less(i, j int) bool {
	return s[i].Rank < s[j].Rank
}

func (s sortByRank) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// NewHandAnalyzer will return a new HandAnalyzer instance
func NewHandAnalyzer(size int, cards []*deck.Card) *HandAnalyzer {
	newCards := make([]*deck.Card, len(cards))
	copy(newCards, cards)

	sort.Sort(sort.Reverse(sortByRank(newCards)))

	h := &HandAnalyzer{
		size:  size,
		cards: newCards,
	}

	h.analyze()

	if _, ok := h.GetFourOfAKind(); ok {
		h.hand = FourOfAKind
	} else if _, ok := h.GetFullHouse(); ok {
		h.hand = FullHouse
	} else if _, ok := h.GetFlush(); ok {
		h.hand = Flush
	} else if _, ok := h.GetThreeOfAKind(); ok {
		h.hand = ThreeOfAKind
	} else if _, ok := h.GetTwoPair(); ok {
		h.hand = TwoPair
	} else if _, ok := h.GetPair(); ok {
		h.hand = OnePair
	} else {
		h.hand = HighCard
	}

	return h
}

// GetHand will return the best possible hand the cards can make
func (h *HandAnalyzer) GetHand() Hand {
	return h.hand
}

// GetFourOfAKind will return the best four of a kind, if possible
func (h *HandAnalyzer) GetFourOfAKind() (int, bool) {
	if len(h.quads) > 0 {
		return h.quads[0], true
	}

	return 0, false
}

// GetThreeOfAKind will return the best three of a kind, if possible
func (h *HandAnalyzer) GetThreeOfAKind() (int, bool) {
	if len(h.trips) > 0 {
		return h.trips[0], true
	}

	return 0, false
}

// GetPair will return the best pair, if possible
func (h *HandAnalyzer) GetPair() (int, bool) {
	if len(h.pairs) > 0 {
		return h.pairs[0], true
	}

	return 0, false
}

// GetTwoPair will return the best two pairs, if possible
func (h *HandAnalyzer) GetTwoPair() ([]int, bool) {
	if len(h.pairs) >= 2 {
		return h.pairs[0:2], true
	}

	return nil, false
}

// GetHighCard will return the high card
func (h *HandAnalyzer) GetHighCard() (int, bool) {
	return h.cards[0].Rank, true
}

// GetFullHouse will return the best full house, if possible
func (h *HandAnalyzer) GetFullHouse() ([]int, bool) {
	if len(h.trips) == 0 {
		return nil, false
	}

	trips := h.trips[0]

	pair, ok := h.GetPair()
	if !ok {
		if len(h.trips) >= 2 {
			pair = h.trips[1]
		} else {
			return nil, false
		}
	}

	if len(h.trips) >= 2 && h.trips[1] > pair {
		pair = h.trips[1]
	}

	return []int{trips, pair}, true
}

// GetFlush will return the best possible flush, if possible
func (h *HandAnalyzer) GetFlush() ([]int, bool) {
	if h.flush != nil {
		return h.flush, true
	}

	return nil, false
}

// GetRoyalFlush will return true if there's a royal flush
func (h *HandAnalyzer) GetRoyalFlush() bool {
	return false
}

// GetStraightFlush will return the best straight flush, if possible
func (h *HandAnalyzer) GetStraightFlush() (int, bool) {
	return 0, false
}

// GetStraight will return the best straight, if possible
func (h *HandAnalyzer) GetStraight() (int, bool) {
	return 0, false
}

func (h *HandAnalyzer) analyze() {
	suitCounts := make(map[deck.Suit][]int)

	prevRank := math.MaxInt8
	numOfRank := 0

	quads := make([]int, 0)
	trips := make([]int, 0)
	pairs := make([]int, 0)

	nCards := len(h.cards)
	for i, card := range h.cards {
		if card.Rank == prevRank {
			numOfRank++
		}

		// if the card is no longer the same rank, or we're at the end
		// check the longest group of cards we can form
		if card.Rank != prevRank || i+1 == nCards {
			switch numOfRank {
			case 4:
				quads = append(quads, prevRank)
			case 3:
				trips = append(trips, prevRank)
			case 2:
				pairs = append(pairs, prevRank)
			}

			numOfRank = 1
		}

		prevRank = card.Rank

		if h.flush == nil {
			ranks, ok := suitCounts[card.Suit]
			if !ok {
				ranks = make([]int, 0, 1)
			}
			ranks = append(ranks, card.Rank)
			suitCounts[card.Suit] = ranks

			if len(ranks) >= h.size {
				h.flush = ranks
			}
		}
	}

	h.quads = quads
	h.trips = trips
	h.pairs = pairs
}
