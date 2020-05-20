package passthepoop

import (
	"fmt"
	"mondaynightpoker-server/pkg/deck"
	"regexp"
	"strconv"
	"strings"
)

var cardRx = regexp.MustCompile(`^(?i)([2-9]|1[0-4])([cdhs])$`)
func card(s string) *deck.Card {
	match := cardRx.FindStringSubmatch(s)
	if match == nil {
		panic(fmt.Sprintf("could not create card from string: %s", s))
	}

	rank, _ := strconv.Atoi(match[1])
	var suit deck.Suit
	switch strings.ToUpper(match[2]) {
	case "C":
		suit = deck.Clubs
	case "D":
		suit = deck.Diamonds
	case "H":
		suit = deck.Hearts
	case "S":
		suit = deck.Spades
	}

	return &deck.Card{
		Rank: rank,
		Suit: suit,
	}
}
