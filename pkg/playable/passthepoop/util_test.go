package passthepoop

import (
	"fmt"
	"mondaynightpoker-server/pkg/deck"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func dealCards(g *Game, cards ...string) {
	for i, c := range cards {
		g.participants[i].card = card(c)
	}
}

func livesEqual(t *testing.T, g *Game, livesMap map[int64]int) {
	t.Helper()

	for id, expectedLives := range livesMap {
		assert.Equal(t, expectedLives, g.idToParticipant[id].lives, "expected player ID %d to have %d lives", id, expectedLives)
	}
}
