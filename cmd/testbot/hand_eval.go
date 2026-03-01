package main

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
)

// cardInfoToDeckCard converts a CardInfo to a *deck.Card using deck.CardFromString.
func cardInfoToDeckCard(c CardInfo) *deck.Card {
	return deck.CardFromString(c.DeckString())
}

// cardInfosToDeckCards converts a slice of CardInfo to []*deck.Card.
func cardInfosToDeckCards(cards []CardInfo) []*deck.Card {
	out := make([]*deck.Card, len(cards))
	for i, c := range cards {
		out[i] = cardInfoToDeckCard(c)
	}
	return out
}

// evaluatePokerHand combines hand and community cards, analyzes as a 5-card poker hand,
// and returns the hand type and numeric strength.
func evaluatePokerHand(hand, community []CardInfo) (handanalyzer.Hand, int) {
	all := make([]CardInfo, 0, len(hand)+len(community))
	all = append(all, hand...)
	all = append(all, community...)
	ha := handanalyzer.New(5, cardInfosToDeckCards(all))
	return ha.GetHand(), ha.GetStrength()
}

// handStrengthScore maps a handanalyzer.Hand to a 0.0–1.0 scale.
func handStrengthScore(h handanalyzer.Hand) float64 {
	switch h {
	case handanalyzer.RoyalFlush:
		return 1.0
	case handanalyzer.StraightFlush:
		return 0.95
	case handanalyzer.FourOfAKind:
		return 0.90
	case handanalyzer.FullHouse:
		return 0.80
	case handanalyzer.Flush:
		return 0.70
	case handanalyzer.Straight:
		return 0.65
	case handanalyzer.ThreeOfAKind:
		return 0.55
	case handanalyzer.TwoPair:
		return 0.40
	case handanalyzer.OnePair:
		return 0.25
	default:
		return 0.10
	}
}

// startingHandStrength returns a pre-flop heuristic strength for a Texas Hold'em starting hand.
func startingHandStrength(hand []CardInfo) float64 {
	if len(hand) < 2 {
		return 0.10
	}

	r1, r2 := hand[0].Rank, hand[1].Rank
	highRank := r1
	if r2 > highRank {
		highRank = r2
	}

	// Pocket pair
	if r1 == r2 {
		return 0.5 + (float64(r1)/14.0)*0.3
	}

	suited := hand[0].Suit == hand[1].Suit
	gap := r1 - r2
	if gap < 0 {
		gap = -gap
	}
	connected := gap == 1

	if suited {
		return 0.25 + (float64(highRank)/14.0)*0.15
	}
	if connected {
		return 0.20 + (float64(highRank)/14.0)*0.15
	}
	return (float64(highRank) / 14.0) * 0.25
}
