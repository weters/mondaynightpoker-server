package deck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Suit represents a card suit
type Suit string

// suit constants
const (
	Hearts   Suit = "hearts"
	Clubs    Suit = "clubs"
	Diamonds Suit = "diamonds"
	Spades   Suit = "spades"
)

// Int returns the integer representation
func (s Suit) Int() int {
	switch s {
	case Clubs:
		return 1
	case Hearts:
		return 1 << 1
	case Diamonds:
		return 1 << 2
	case Spades:
		return 1 << 3
	}

	return 0
}

// Card is an individual playing card
type Card struct {
	Rank int  `json:"rank"`
	Suit Suit `json:"suit"`
}

// face cards
const (
	Jack    = 11
	Queen   = 12
	King    = 13
	Ace     = 14
	HighAce = Ace
	LowAce  = 1
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

// AceLowRank return the rank where Ace is considered low instead of high
func (c *Card) AceLowRank() int {
	if c.Rank == Ace {
		return 1
	}

	return c.Rank
}

var cardRx = regexp.MustCompile(`(?i)^([0-9]|1[0-4])([cdhs])\z`)

// CardFromString returns a Card from the string.
// The string must be in the format of <rank><suit> where rank >= 2 and <= 14 and suit in [cdhs]
func CardFromString(s string) *Card {
	match := cardRx.FindStringSubmatch(s)
	if match == nil {
		panic(fmt.Sprintf("could not parse card: %s", s))
	}

	rank, err := strconv.Atoi(match[1])
	if err != nil {
		panic(fmt.Sprintf("could not parse card `%s`: %v", s, err))
	}

	var suit Suit
	switch strings.ToLower(match[2]) {
	case "c":
		suit = Clubs
	case "d":
		suit = Diamonds
	case "h":
		suit = Hearts
	case "s":
		suit = Spades
	default:
		// should never be hit due to the regexp
		panic("unknown suit")
	}

	return &Card{
		Rank: rank,
		Suit: suit,
	}
}

// CardsFromString will returns a slice of cards
func CardsFromString(s string) []*Card {
	cardStrings := strings.Split(s, ",")
	cards := make([]*Card, len(cardStrings))
	for i, card := range cardStrings {
		cards[i] = CardFromString(card)
	}

	return cards
}

// CardsToString will convert a slice of cards to a string in the format of 2c,3h,4s,...
func CardsToString(cards []*Card) string {
	c := make([]string, len(cards))
	for i, card := range cards {
		var suit string
		switch card.Suit {
		case Clubs:
			suit = "c"
		case Hearts:
			suit = "h"
		case Diamonds:
			suit = "d"
		case Spades:
			suit = "s"
		}

		c[i] = fmt.Sprintf("%d%s", card.Rank, suit)
	}

	return strings.Join(c, ",")
}
