package bourre

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"strings"
	"testing"
)

func TestPlayer_PlayCard(t *testing.T) {
	p := &Player{
		hand: cardsFromString("2c,3c,4c,5c"),
	}

	assert.Equal(t, ErrCardNotInPlayersHand, p.PlayCard(cardFromString("8c")))
	assert.NoError(t, p.PlayCard(cardFromString("3c")))
	assert.Equal(t, "2c,4c,5c", cardsToString(p.hand))
}

func cardsToString(cards []*deck.Card) string {
	sArray := make([]string, len(cards))
	for i, card := range cards {
		var suit string
		switch card.Suit {
		case deck.Clubs:
			suit = "c"
		case deck.Diamonds:
			suit = "d"
		case deck.Hearts:
			suit = "h"
		case deck.Spades:
			suit = "s"
		}

		sArray[i] = fmt.Sprintf("%d%s", card.Rank, suit)
	}

	return strings.Join(sArray, ",")
}
