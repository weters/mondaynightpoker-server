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

// Card is an individual playing card
type Card struct {
	Rank   int  `json:"rank"`
	Suit   Suit `json:"suit"`
	IsWild bool `json:"isWild"`
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

var cardRx = regexp.MustCompile(`(?i)^(!)?([0-9]|1[0-4])([cdhs])\z`)

// CardFromString returns a Card from the string.
// The string must be in the format of <rank><suit> where rank >= 2 and <= 14 and suit in [cdhs]
func CardFromString(s string) *Card {
	if s == "" {
		return nil
	}

	match := cardRx.FindStringSubmatch(s)
	if match == nil {
		panic(fmt.Sprintf("could not parse card: %s", s))
	}

	isWild := match[1] == "!"

	rank, err := strconv.Atoi(match[2])
	if err != nil {
		panic(fmt.Sprintf("could not parse card `%s`: %v", s, err))
	}

	var suit Suit
	switch strings.ToLower(match[3]) {
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
		Rank:   rank,
		Suit:   suit,
		IsWild: isWild,
	}
}

// CardsFromString will returns a slice of cards
func CardsFromString(s string) []*Card {
	if s == "" {
		return []*Card{}
	}

	cardStrings := strings.Split(s, ",")
	cards := make([]*Card, len(cardStrings))
	for i, card := range cardStrings {
		cards[i] = CardFromString(card)
	}

	return cards
}

// CardToString converts a card (Ace of Clubs) to a string (14c)
func CardToString(card *Card) string {
	if card == nil {
		return ""
	}

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

	isWild := ""
	if card.IsWild {
		isWild = "!"
	}

	return fmt.Sprintf("%s%d%s", isWild, card.Rank, suit)
}

// CardsToString will convert a slice of cards to a string in the format of 2c,3h,4s,...
func CardsToString(cards []*Card) string {
	c := make([]string, len(cards))
	for i, card := range cards {
		c[i] = CardToString(card)
	}

	return strings.Join(c, ",")
}
