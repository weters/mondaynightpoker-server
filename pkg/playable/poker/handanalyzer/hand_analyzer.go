package handanalyzer

import (
	"math"
	"mondaynightpoker-server/pkg/deck"
	"sort"
)

// HandAnalyzer can analyze a hand
type HandAnalyzer struct {
	size          int
	cards         deck.Hand
	wildCards     deck.Hand
	flush         []int
	quads         []int
	trips         []int
	pairs         []int
	straightFlush int
	straight      int

	hand     Hand
	strength int

	// assignment tracks wild mode assignments for constrained wild analysis
	assignment []WildAssignment
}

// New will return a new HandAnalyzer instance
func New(size int, cards []*deck.Card) *HandAnalyzer {
	// clone to prevent modifying original
	sortedCards := make(deck.Hand, len(cards))
	copy(sortedCards, cards)
	sort.Sort(sort.Reverse(sortByRank(sortedCards)))

	nonWilds := make(deck.Hand, 0, len(sortedCards))
	wilds := make(deck.Hand, 0, len(sortedCards))

	for _, card := range sortedCards {
		if card.IsWild {
			wilds.AddCard(card)
		} else {
			nonWilds.AddCard(card)
		}
	}

	var best *HandAnalyzer
	if len(wilds) == 0 {
		// No wilds - use existing fast path
		best = &HandAnalyzer{
			size:      size,
			cards:     nonWilds,
			wildCards: wilds,
		}
		best.analyzeHand()
	} else {
		// Try all 2^N wild mode combinations and pick the best
		best = findBestConstrainedHand(size, nonWilds, wilds)
	}

	best.calculateHand()
	return best
}

// findBestConstrainedHand tries all wild mode combinations and returns the best hand
func findBestConstrainedHand(size int, nonWilds, wilds deck.Hand) *HandAnalyzer {
	combinations := generateWildCombinations(wilds)
	var best *HandAnalyzer
	bestStrength := -1

	for _, assignment := range combinations {
		h := analyzeWithAssignment(size, nonWilds, wilds, assignment)
		h.calculateHand()
		if s := h.GetStrength(); s > bestStrength {
			bestStrength = s
			best = h
		}
	}
	return best
}

// analyzeWithAssignment analyzes a hand with a specific wild mode assignment
func analyzeWithAssignment(size int, nonWilds, wilds deck.Hand, assignment []WildAssignment) *HandAnalyzer {
	h := &HandAnalyzer{
		size:       size,
		cards:      nonWilds,
		wildCards:  wilds,
		assignment: assignment,
	}

	h.analyzeHandConstrained()
	return h
}

// analyzeHandConstrained analyzes a hand with constrained wild cards
func (h *HandAnalyzer) analyzeHandConstrained() {
	// keeps track of flushes
	suitCounts := make(map[deck.Suit][]int)

	// keeps track of pairs, trips, and quads
	prevRank := math.MaxInt8
	numOfRank := 1

	nCards := len(h.cards)
	for i, card := range h.cards {
		if h.flush == nil {
			h.checkFlushConstrained(card, suitCounts)
		}

		isLastCard := i+1 == nCards
		h.checkPairs(card, isLastCard, &prevRank, &numOfRank)
	}

	h.updatePairsWithWildsConstrained()

	// Check straights using the simplified algorithm
	// Try high ace first
	h.straight = checkStraightSimple(h.cards, h.assignment, h.size, false, "", deck.HighAce)
	if h.straight == 0 {
		// Try low ace (wheel)
		h.straight = checkStraightSimple(h.cards, h.assignment, h.size, false, "", deck.LowAce)
	}

	// Check straight flushes for each suit
	for _, suit := range []deck.Suit{deck.Clubs, deck.Diamonds, deck.Hearts, deck.Spades} {
		// Try high ace
		sf := checkStraightSimple(h.cards, h.assignment, h.size, true, suit, deck.HighAce)
		if sf > h.straightFlush {
			h.straightFlush = sf
		}
		// Try low ace
		sf = checkStraightSimple(h.cards, h.assignment, h.size, true, suit, deck.LowAce)
		if sf > h.straightFlush {
			h.straightFlush = sf
		}
	}
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
	numOfRank := 1

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

	h.updatePairsWithWilds()

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

	// it's impossible for a full-house with two wilds as you can
	// make a better hand
	if len(h.wildCards) >= 2 {
		return nil, false
	}

	if h.size < 5 || len(h.cards)+len(h.wildCards) < 5 {
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
		// in an 8-card hand, we may have two sets of trips and a separate pair
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
	if h.size < 5 {
		return nil, false
	}

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

		// if we have an extra wild, it becomes an Ace kicker
		if len(h.wildCards) > 0 && found+len(h.wildCards) > 4 {
			hc = deck.Ace
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

	nWilds := len(h.wildCards)
	if len(ranks)+nWilds >= h.size {
		if nWilds > 0 {
			wilds := make([]int, nWilds)
			for i := range wilds {
				wilds[i] = deck.Ace
			}

			ranks = append(wilds, ranks...)
		}

		h.flush = ranks[0:h.size]
	}
}

// checkFlushConstrained checks for a flush with constrained wild cards
// Suit-mode wilds can contribute to ANY suit
// Rank-mode wilds can ONLY contribute to their original suit
func (h *HandAnalyzer) checkFlushConstrained(card *deck.Card, suitCounts map[deck.Suit][]int) {
	ranks, ok := suitCounts[card.Suit]
	if !ok {
		ranks = make([]int, 0, 1)
	}
	ranks = append(ranks, card.Rank)
	suitCounts[card.Suit] = ranks

	// Count wilds that can contribute to this suit:
	// - Suit-mode wilds: can contribute to ANY suit (change suit, keep rank)
	//   BUT only one wild per rank can contribute (can't have two 2♦ in a deck)
	// - Rank-mode wilds: can ONLY contribute to their original suit (change rank, keep suit)
	suitModeWilds := getSuitModeWilds(h.assignment)
	rankModeWildsForSuit := getRankModeWildsForSuit(h.assignment, card.Suit)

	// Build unique suit-mode wild ranks (deduplicate same-rank wilds)
	// Also exclude ranks that are already in the flush from non-wilds
	suitModeRanks := make(map[int]bool)
	existingRanks := make(map[int]bool)
	for _, r := range ranks {
		existingRanks[r] = true
	}
	for _, wa := range suitModeWilds {
		if !existingRanks[wa.Card.Rank] && !suitModeRanks[wa.Card.Rank] {
			suitModeRanks[wa.Card.Rank] = true
		}
	}

	availableWilds := len(suitModeRanks) + len(rankModeWildsForSuit)

	if len(ranks)+availableWilds >= h.size {
		// Build the flush with wilds contributing
		wildRanks := make([]int, 0, availableWilds)

		// Add unique suit-mode wild ranks
		for r := range suitModeRanks {
			wildRanks = append(wildRanks, r)
		}

		// Add rank-mode wilds as aces (they can be any rank)
		for range rankModeWildsForSuit {
			wildRanks = append(wildRanks, deck.Ace)
		}

		// Sort wilds by rank descending
		sort.Sort(sort.Reverse(sort.IntSlice(wildRanks)))

		// Merge sorted wilds with sorted ranks
		allRanks := make([]int, 0, len(wildRanks)+len(ranks))
		wi, ri := 0, 0
		for wi < len(wildRanks) || ri < len(ranks) {
			if wi >= len(wildRanks) {
				allRanks = append(allRanks, ranks[ri])
				ri++
			} else if ri >= len(ranks) {
				allRanks = append(allRanks, wildRanks[wi])
				wi++
			} else if wildRanks[wi] >= ranks[ri] {
				allRanks = append(allRanks, wildRanks[wi])
				wi++
			} else {
				allRanks = append(allRanks, ranks[ri])
				ri++
			}
		}

		h.flush = allRanks[0:h.size]
	}
}

func (h *HandAnalyzer) checkPairs(card *deck.Card, isLastCard bool, prevRank, numOfRank *int) {
	if card.Rank == *prevRank {
		*numOfRank++
	}

	// if the card is no longer the same rank, or we're at the end
	// check the longest group of cards we can form
	// Note: the first check in conditional is to make sure we skip the first
	// card
	if card.Rank != *prevRank || isLastCard {
		// make sure this isn't the first card
		if *prevRank != math.MaxInt8 || isLastCard {
			// this may happen if all our cards except one is a wild
			if isLastCard && *prevRank == math.MaxInt8 {
				*prevRank = card.Rank
			}

			num := *numOfRank
			if num > 4 {
				num = 4
			}

			switch num {
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
		}

		// reset back to 1 since we changed rank
		*numOfRank = 1
	}

	*prevRank = card.Rank
}

func getBestRank(ranks []int) int {
	if len(ranks) > 0 {
		return ranks[0]
	}

	return math.MinInt32
}

// distributeWildsForPairs distributes wilds to optimize pairs/trips/quads
func (h *HandAnalyzer) distributeWildsForPairs(wildCount int, quadRank, tripsRank, pairsRank, highCard int) {
	switch wildCount {
	case 3:
		if quadRank > tripsRank && quadRank > pairsRank && quadRank >= highCard {
			return
		} else if tripsRank > pairsRank && tripsRank >= highCard {
			h.trips = h.trips[1:]
			h.quads = []int{tripsRank}
		} else if pairsRank > tripsRank && pairsRank >= highCard {
			h.pairs = h.pairs[1:]
			h.quads = []int{pairsRank}
		} else if highCard > 0 {
			h.quads = []int{highCard}
		} else {
			h.trips = []int{deck.Ace}
		}
	case 2:
		if quadRank > tripsRank && quadRank > pairsRank {
			return
		} else if tripsRank > pairsRank {
			h.quads = []int{tripsRank}
			h.trips = h.trips[1:]
		} else if pairsRank > 0 {
			h.quads = []int{pairsRank}
			h.pairs = h.pairs[1:]
		} else if highCard > 0 {
			h.trips = []int{highCard}
		} else {
			h.pairs = []int{deck.Ace}
		}
	case 1:
		if quadRank > tripsRank {
			return
		} else if tripsRank > 0 {
			h.quads = []int{tripsRank}
			h.trips = h.trips[1:]
		} else if pairsRank > 0 {
			h.trips = []int{pairsRank}
			h.pairs = h.pairs[1:]
		} else if highCard > 0 {
			h.pairs = []int{highCard}
		}
	}
}

func (h *HandAnalyzer) updatePairsWithWilds() {
	nWilds := len(h.wildCards)
	if nWilds == 0 {
		return
	}

	quadRank := getBestRank(h.quads)
	tripsRank := getBestRank(h.trips)
	pairsRank := getBestRank(h.pairs)
	highCard := 0
	if len(h.cards) > 0 {
		highCard = h.cards[0].Rank
	}

	// make four-of-a-kind
	if nWilds >= 4 {
		h.quads = []int{deck.Ace}
		return
	}

	h.distributeWildsForPairs(nWilds, quadRank, tripsRank, pairsRank, highCard)
}

// updatePairsWithWildsConstrained updates pairs/trips/quads with constrained wild cards
// Rank-mode wilds: flexible, can match any rank (like current behavior)
// Suit-mode wilds: can ONLY count toward their original rank
func (h *HandAnalyzer) updatePairsWithWildsConstrained() {
	if len(h.assignment) == 0 {
		return
	}

	// First, count suit-mode wilds and add them to their fixed rank counts
	// This is done by re-examining the cards with suit-mode wilds included
	rankCounts := make(map[int]int)

	// Count non-wild cards
	for _, card := range h.cards {
		rankCounts[card.Rank]++
	}

	// Add suit-mode wilds to their fixed rank
	for _, wa := range h.assignment {
		if wa.Mode == SuitMode {
			rankCounts[wa.Card.Rank]++
		}
	}

	// Count rank-mode wilds (these are flexible)
	rankModeWildCount := 0
	for _, wa := range h.assignment {
		if wa.Mode == RankMode {
			rankModeWildCount++
		}
	}

	// Rebuild pairs, trips, quads from the updated rank counts
	h.quads = nil
	h.trips = nil
	h.pairs = nil

	// Get all ranks sorted by count descending, then by rank descending
	type rankInfo struct {
		rank  int
		count int
	}
	ranksInfo := make([]rankInfo, 0, len(rankCounts))
	for rank, count := range rankCounts {
		if count > 4 {
			count = 4 // cap at four-of-a-kind
		}
		ranksInfo = append(ranksInfo, rankInfo{rank, count})
	}

	// Sort by count descending, then rank descending
	sort.Slice(ranksInfo, func(i, j int) bool {
		if ranksInfo[i].count != ranksInfo[j].count {
			return ranksInfo[i].count > ranksInfo[j].count
		}
		return ranksInfo[i].rank > ranksInfo[j].rank
	})

	// Categorize the rank counts
	for _, ri := range ranksInfo {
		switch ri.count {
		case 4:
			h.quads = append(h.quads, ri.rank)
		case 3:
			h.trips = append(h.trips, ri.rank)
		case 2:
			h.pairs = append(h.pairs, ri.rank)
		}
	}

	// Now distribute rank-mode wilds optimally
	if rankModeWildCount == 0 {
		return
	}

	quadRank := getBestRank(h.quads)
	tripsRank := getBestRank(h.trips)
	pairsRank := getBestRank(h.pairs)
	highCard := 0
	if len(h.cards) > 0 {
		highCard = h.cards[0].Rank
	}

	// Special case: four or more rank-mode wilds
	if rankModeWildCount >= 4 {
		h.quads = []int{deck.Ace}
		return
	}

	h.distributeWildsForPairs(rankModeWildCount, quadRank, tripsRank, pairsRank, highCard)
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
