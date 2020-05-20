package deck

import (
	"fmt"
	"strconv"
)

// Suit represents a card suit
type Suit string

// suit constants
const (
	Hearts Suit = "hearts"
	Clubs Suit = "clubs"
	Diamonds Suit = "diamonds"
	Spades Suit = "spades"
)

// Card is an individual playing card
type Card struct {
	Rank int `json:"rank"`
	Suit Suit `json:"suit"`
}

// face cards
const (
	Jack = 11
	Queen = 12
	King = 13
	Ace =14
)

func (c *Card) String() string {
	var rank string
	switch c.Rank {
	case Jack:
		rank = "J"
	case Queen:
		rank = "Q"
	case King:
		rank = "K"
	case Ace:
		rank = "A"
	default:
		rank = strconv.Itoa(c.Rank)
	}

	var suit string
	switch c.Suit {
	case Clubs:
		suit = "♣"
	case Diamonds:
		suit = "♢"
	case Hearts:
		suit = "♡"
	case Spades:
		suit = "♠"
	default:
		panic("unknown suit")
	}

	return fmt.Sprintf("%s%s", rank, suit)
}

// Equal returns true if the cards are equal (matches suit and rank)
func (c *Card) Equal(card *Card) bool {
	return c.Suit == card.Suit && c.Rank == card.Rank
}
