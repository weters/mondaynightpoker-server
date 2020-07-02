package poker

import "mondaynightpoker-server/pkg/deck"

// used to keep track of the straight progress
type straightTracker struct {
	startRank int
	prevRank  int
	streak    int
}

// checkStraight will check for a straight
// If one has been found, then the highest card in the straight will be assigned to the "val"
func (h *HandAnalyzer) checkStraight(card *deck.Card, st *straightTracker, aceValue int, val *int) {
	cardRank := card.Rank
	if card.Rank == deck.Ace && aceValue == deck.LowAce {
		cardRank = deck.LowAce
	}

	inStraight := false
	if cardRank+1 == st.prevRank {
		inStraight = true
		st.streak++
	} else if cardRank == st.prevRank {
		inStraight = true
	}

	if st.streak >= h.size {
		*val = st.startRank
	}

	if !inStraight {
		st.streak = 1
		st.startRank = cardRank
	}

	st.prevRank = cardRank
}
