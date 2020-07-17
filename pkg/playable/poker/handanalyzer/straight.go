package handanalyzer

import "mondaynightpoker-server/pkg/deck"

// used to keep track of the straight progress
type straightTracker struct {
	streak deck.Hand
}

// checkStraight will check for a straight
// If one has been found, then the highest card in the straight will be assigned to the "val"
func (h *HandAnalyzer) checkStraight(card *deck.Card, st *straightTracker, aceValue int, val *int) {
	cardRank := card.Rank
	if card.Rank == deck.Ace && aceValue == deck.LowAce {
		cardRank = deck.LowAce
	}

	if len(st.streak) == 0 {
		st.streak = deck.Hand{card}
		return
	}

	lastCard := st.streak.LastCard()
	diffInRank := lastCard.Rank - cardRank
	if diffInRank == 0 {
		// same rank
		return
	} else if diffInRank == 1 {
		st.streak.AddCard(card)
	} else {
		st.streak = deck.Hand{card}
	}

	if len(st.streak) >= h.size {
		firstCard := st.streak.FirstCard()
		rank := firstCard.Rank
		if firstCard.Rank == deck.Ace && aceValue == deck.LowAce {
			rank = deck.LowAce
		}
		*val = rank
	}
}
