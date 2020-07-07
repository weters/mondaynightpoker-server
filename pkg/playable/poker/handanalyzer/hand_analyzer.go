package handanalyzer

import (
	"math"
	"mondaynightpoker-server/pkg/deck"
	"sort"
)

// HandAnalyzer can analyze a hand
type HandAnalyzer struct {
	size          int
	cards         []*deck.Card
	flush         []int
	quads         []int
	trips         []int
	pairs         []int
	straightFlush int
	straight      int

	hand     Hand
	strength int
}

// New will return a new HandAnalyzer instance
func New(size int, cards []*deck.Card) *HandAnalyzer {
	newCards := make([]*deck.Card, len(cards))
	copy(newCards, cards)

	sort.Sort(sort.Reverse(sortByRank(newCards)))

	h := &HandAnalyzer{
		size:  size,
		cards: newCards,
	}

	// the method order here is required
	h.analyzeHand()
	h.calculateHand()

	return h
}

// analyzeHand will loop through a players hand and calculate the various combinations
// This is required to be called in order for the public Get*() methods to return properly
// This method should only be called once from the constructor
func (h *HandAnalyzer) analyzeHand() {
	// keeps track of flushes
	suitCounts := make(map[deck.Suit][]int)

	// straight-flush tracker
	sfTracker := map[deck.Suit]*straightTracker{
		deck.Clubs:    {},
		deck.Diamonds: {},
		deck.Hearts:   {},
		deck.Spades:   {},
	}

	// straight tracker
	sTracker := straightTracker{}

	// keeps track of pairs, trips, and quads
	prevRank := math.MaxInt8
	numOfRank := 0

	nCards := len(h.cards)
	for i, card := range h.cards {
		if h.straightFlush == 0 {
			h.checkStraight(card, sfTracker[card.Suit], deck.HighAce, &h.straightFlush)
		}

		if h.straight == 0 {
			h.checkStraight(card, &sTracker, deck.HighAce, &h.straight)
		}

		if h.flush == nil {
			h.checkFlush(card, suitCounts)
		}

		isLastCard := i+1 == nCards
		h.checkPairs(card, isLastCard, &prevRank, &numOfRank)
	}

	// check for straights and straight-flushes with a low-ace
	for _, card := range h.cards {
		if card.Rank != deck.Ace {
			break
		}

		if h.straightFlush == 0 {
			h.checkStraight(card, sfTracker[card.Suit], deck.LowAce, &h.straightFlush)
		}

		if h.straight == 0 {
			h.checkStraight(card, &sTracker, deck.LowAce, &h.straight)
		}
	}
}

// GetHand will return the best possible hand the cards can make
func (h *HandAnalyzer) GetHand() Hand {
	return h.hand
}

// GetRoyalFlush will return true if there's a royal flush
func (h *HandAnalyzer) GetRoyalFlush() bool {
	return h.straightFlush > 0 && h.straightFlush == deck.Ace
}

// GetStraightFlush will return the best straight flush, if possible
func (h *HandAnalyzer) GetStraightFlush() (int, bool) {
	if h.straightFlush > 0 {
		return h.straightFlush, true
	}

	return 0, false
}

// GetFourOfAKind will return the best four of a kind, if possible
func (h *HandAnalyzer) GetFourOfAKind() (int, bool) {
	// cannot get four-of-a-kind in 3 card poker
	if h.size < 4 {
		return 0, false
	}

	if len(h.quads) > 0 {
		return h.quads[0], true
	}

	return 0, false
}

// GetFullHouse will return the best full house, if possible
func (h *HandAnalyzer) GetFullHouse() ([]int, bool) {
	if len(h.trips) == 0 {
		return nil, false
	}

	if h.size < 5 {
		return nil, false
	}

	trips := h.trips[0]

	pair, ok := h.GetPair()
	if !ok {
		if len(h.trips) == 1 {
			// could not find a pair from a second set of trips
			return nil, false
		}

		pair = h.trips[1]
	} else if len(h.trips) >= 2 && h.trips[1] > pair {
		// in an 8-card hand, we may have two sets of trips and a sepaarate pair
		// in that case, let's make sure we grab the better pair from the trips
		pair = h.trips[1]
	}

	return []int{trips, pair}, true
}

func (h *HandAnalyzer) getThreeCardPokerThreeOfAKind() (int, bool) {
	if h.size > 3 {
		return 0, false
	}

	if len(h.quads) > 0 {
		return h.quads[0], true
	}

	return h.GetThreeOfAKind()
}

func (h *HandAnalyzer) getThreeCardPokerStraight() (int, bool) {
	if h.size > 3 {
		return 0, false
	}

	return h.GetStraight()
}

// GetFlush will return the best possible flush, if possible
func (h *HandAnalyzer) GetFlush() ([]int, bool) {
	if h.flush != nil {
		return h.flush, true
	}

	return nil, false
}

// GetStraight will return the best straight, if possible
func (h *HandAnalyzer) GetStraight() (int, bool) {
	if h.straight > 0 {
		return h.straight, true
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

// GetTwoPair will return the best two pairs, if possible
func (h *HandAnalyzer) GetTwoPair() ([]int, bool) {
	if len(h.pairs) >= 2 {
		return h.pairs[0:2], true
	}

	return nil, false
}

// GetPair will return the best pair, if possible
func (h *HandAnalyzer) GetPair() (int, bool) {
	if len(h.pairs) > 0 {
		return h.pairs[0], true
	}

	return 0, false
}

// GetHighCard will return the high card
func (h *HandAnalyzer) GetHighCard() ([]int, bool) {
	cards := make([]int, h.size)
	for i := 0; i < h.size; i++ {
		if i < len(h.cards) {
			cards[i] = h.cards[i].Rank
		}
	}
	return cards, true
}

func calculateStrength(hand Hand, cards []int) int {
	fiveCards := make([]int, 5)
	copy(fiveCards, cards)

	strength := math.Pow(15, 5) * float64(hand)
	for i := 0; i < 5; i++ {
		val := fiveCards[4-i]
		strength += math.Pow(15, float64(i)) * float64(val)
	}

	return int(strength)
}

// GetStrength returns the strength of the hand
func (h *HandAnalyzer) GetStrength() int {
	if h.strength > 0 {
		return h.strength
	}

	h.strength = h.getStrength()
	return h.strength
}

func (h *HandAnalyzer) getStrength() int {
	hand := h.GetHand()

	switch hand {
	case HighCard:
		c, _ := h.GetHighCard()
		return calculateStrength(hand, c)
	case OnePair:
		pair, _ := h.GetPair()
		hc := make([]int, 0)
		for _, card := range h.cards {
			if card.Rank == pair {
				continue
			}

			hc = append(hc, card.Rank)
			if len(hc) == h.size-2 {
				break
			}
		}
		return calculateStrength(hand, append([]int{pair}, hc...))
	case TwoPair:
		twoPair, _ := h.GetTwoPair()
		hc := 0
		for _, card := range h.cards {
			if card.Rank == twoPair[0] || card.Rank == twoPair[1] {
				continue
			}

			hc = card.Rank
			break
		}
		return calculateStrength(hand, []int{twoPair[0], twoPair[1], hc})
	case ThreeOfAKind:
		trips, _ := h.GetThreeOfAKind()
		hc := make([]int, 0)
		for _, card := range h.cards {
			if card.Rank == trips {
				continue
			}

			hc = append(hc, card.Rank)
			if len(hc) >= 2 {
				break
			}
		}
		return calculateStrength(hand, append([]int{trips}, hc...))
	case Straight:
		s, _ := h.GetStraight()
		return calculateStrength(hand, []int{s})
	case Flush:
		f, _ := h.GetFlush()
		return calculateStrength(hand, f)
	case ThreeCardPokerStraight:
		s, _ := h.getThreeCardPokerStraight()
		return calculateStrength(hand, []int{s})
	case ThreeCardPokerThreeOfAKind:
		t, _ := h.getThreeCardPokerThreeOfAKind()
		return calculateStrength(hand, []int{t})
	case FullHouse:
		fh, _ := h.GetFullHouse()
		return calculateStrength(hand, fh)
	case FourOfAKind:
		fk, _ := h.GetFourOfAKind()
		found := 0
		hc := 0
		for _, c := range h.cards {
			if c.Rank == fk {
				found++
				if found <= 4 {
					continue
				}
			}

			hc = c.Rank
			break
		}

		return calculateStrength(hand, []int{fk, hc})
	case StraightFlush:
		s, _ := h.GetStraightFlush()
		return calculateStrength(hand, []int{s})
	case RoyalFlush:
		return calculateStrength(hand, []int{})
	}

	panic("unknown hand")
}

func (h *HandAnalyzer) checkFlush(card *deck.Card, suitCounts map[deck.Suit][]int) {
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

func (h *HandAnalyzer) checkPairs(card *deck.Card, isLastCard bool, prevRank, numOfRank *int) {
	if card.Rank == *prevRank {
		*numOfRank++
	}

	// if the card is no longer the same rank, or we're at the end
	// check the longest group of cards we can form
	if card.Rank != *prevRank || isLastCard {
		switch *numOfRank {
		case 5:
			fallthrough
		case 4:
			if h.quads == nil {
				h.quads = make([]int, 0, 1)
			}

			h.quads = append(h.quads, *prevRank)
		case 3:
			if h.trips == nil {
				h.trips = make([]int, 0, 1)
			}

			h.trips = append(h.trips, *prevRank)
		case 2:
			if h.pairs == nil {
				h.pairs = make([]int, 0, 1)
			}

			h.pairs = append(h.pairs, *prevRank)
		}

		*numOfRank = 1
	}

	*prevRank = card.Rank
}

// calculateHand will determine the best hand
// This must be called after analyzeHand() has been called
func (h *HandAnalyzer) calculateHand() {
	if h.GetRoyalFlush() {
		h.hand = RoyalFlush
	} else if _, ok := h.GetStraightFlush(); ok {
		h.hand = StraightFlush
	} else if _, ok := h.GetFourOfAKind(); ok {
		h.hand = FourOfAKind
	} else if _, ok := h.GetFullHouse(); ok {
		h.hand = FullHouse
	} else if _, ok := h.getThreeCardPokerThreeOfAKind(); ok {
		// in three card poker, a three-of-a-kind is better than straight and flush
		h.hand = ThreeCardPokerThreeOfAKind
	} else if _, ok := h.getThreeCardPokerStraight(); ok {
		// in three card poker, a straight is better than a flush
		h.hand = ThreeCardPokerStraight
	} else if _, ok := h.GetFlush(); ok {
		h.hand = Flush
	} else if _, ok := h.GetStraight(); ok {
		h.hand = Straight
	} else if _, ok := h.GetThreeOfAKind(); ok {
		h.hand = ThreeOfAKind
	} else if _, ok := h.GetTwoPair(); ok {
		h.hand = TwoPair
	} else if _, ok := h.GetPair(); ok {
		h.hand = OnePair
	} else {
		h.hand = HighCard
	}
}
