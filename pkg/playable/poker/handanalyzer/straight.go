package handanalyzer

import (
	"mondaynightpoker-server/pkg/deck"
)

// used to keep track of the straight progress
type straightTracker struct {
	streak    deck.Hand
	usedWilds int
}

func (s *straightTracker) resetWithCard(card *deck.Card) {
	s.streak = deck.Hand{card}
	s.usedWilds = 0
}

// reclaimWilds will remove all cards through the first set of
// wilds found in the streak. For example, if the streak currently contains
// 9c, !8d, !7h, 6s; then this method will remove the first three cards
//
// Important! this method assumes that the last card in the streak is never
// a wild
func (s *straightTracker) reclaimWilds(wildsInHand, currentGap int) {
	if wildsInHand < currentGap {
		panic("reclaimWilds() must only be called if we can reclaim enough wilds to cover the gap")
	}

	if s.streak.LastCard().IsWild {
		panic("reclaimWilds() must not be called if the last card in the streak is a wild")
	}

	foundWild := 0
	maxI := 0

	for i, card := range s.streak {
		maxI = i
		if card.IsWild {
			foundWild++
		} else if foundWild > 0 {
			// stop here
			break
		}
	}

	s.streak = s.streak[maxI:]
	s.usedWilds -= foundWild
	unusedWilds := wildsInHand - s.usedWilds
	if unusedWilds < currentGap {
		// since the streak cannot start or end with a wild, if we have a non-wild card between two
		// groups of wild, we at a minimum have a five-card straight. Therefore, we should never
		// have to free up more than one group of wilds. If this code is hit, we screwed up elsewhere
		// (perhaps we are trying to reclaim cards when a straight has already been found)
		panic("impossible 3 or 5-card streak found")
	}
}

// checkStraight will check for a straight
// If one has been found, then the highest card in the straight will be assigned to the "val"
func (h *HandAnalyzer) checkStraight(card *deck.Card, st *straightTracker, aceValue int, val *int) {
	cardRank := card.Rank
	if cardRank == deck.Ace && aceValue == deck.LowAce {
		cardRank = deck.LowAce
	}

	// currently no streak, so we start from scratch
	if len(st.streak) == 0 {
		st.resetWithCard(card)
		return
	}

	lastCard := st.streak.LastCard()
	diffInRank := lastCard.Rank - cardRank // 8C - 6H = diff of 2
	gapBetweenRanks := diffInRank - 1      // 8C - 6H = gap of 1
	wildsInHand := len(h.wildCards)

	if diffInRank == 0 {
		// same rank
		return
	} else if diffInRank == 1 {
		// we found the next card in a straight
		st.streak.AddCard(card)
	} else if gapBetweenRanks > wildsInHand {
		// the gap between the previous card and the current card cannot be filled in
		// with wilds
		st.resetWithCard(card)
	} else {
		// if we are here, there's at least a gap between the previous card and the current
		// card that one or more wilds can fill.

		// check if there are any unused wilds that we can still use, if not, we need to
		// reclaim the wilds from earlier in the streak
		// i.e., card=5c, and the streak is Jc,W,W,8d
		// in this case, A straight containing J–8 is impossible, so let's see if we can
		// make one with 8–5
		unusedWilds := wildsInHand - st.usedWilds
		if unusedWilds < gapBetweenRanks {
			st.reclaimWilds(wildsInHand, gapBetweenRanks)
			unusedWilds = wildsInHand - st.usedWilds
		}

		if unusedWilds >= gapBetweenRanks {
			// as long as reclaimWilds() worked, this `if` shall always be hit
			for i := 0; i < gapBetweenRanks; i++ {
				st.streak.AddCard(&deck.Card{IsWild: true, Rank: cardRank - i - 1})
				st.usedWilds++
			}
		} else {
			// This line is literally impossible to cover provided
			// the earlier `if gapBetweenRanks > wildsInHand` is in place
			// and reclaimWilds() did its trick
			// Keeping this panic here to safeguard against future stupidity
			panic("unusedWilds were not freed up properly")
		}

		st.streak.AddCard(card)
	}

	// we know if we have a straight if the length of our current streak + any unused
	// wilds is at our threshold
	unusedWilds := len(h.wildCards) - st.usedWilds
	calculatedStreak := len(st.streak) + unusedWilds
	if calculatedStreak >= h.size {
		firstCard := st.streak.FirstCard()
		rank := firstCard.Rank

		// if we have unused wilds, add them to the start of our streak (make it higher),
		// but make sure we do not exceed the rank of an Ace
		rank += unusedWilds
		if rank > deck.Ace {
			rank = deck.Ace
		}

		*val = rank
	}
}
