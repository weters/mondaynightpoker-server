package handanalyzer

import "fmt"

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
	ThreeCardPokerStraight     // in three-card poker, straight beats flush
	ThreeCardPokerThreeOfAKind // in three-card poker, beats straight and flush
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
	case ThreeCardPokerThreeOfAKind:
		return "Three of a kind"
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
