package bourre

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"strings"
	"testing"
)

func TestPlayer_PlayCard(t *testing.T) {
	p := &Player{
		hand: cardsFromString("2c,3c,4c,5c"),
	}

	assert.Equal(t, ErrCardNotInPlayersHand, p.playerDidPlayCard(cardFromString("8c")))
	assert.NoError(t, p.playerDidPlayCard(cardFromString("3c")))
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

func TestPlayer_GetValidMoves(t *testing.T) {
	a := assert.New(t)

	player1 := NewPlayer(1)
	player2 := NewPlayer(2)

	g, err := newGame(logrus.StandardLogger(), []*Player{player1, player2}, nil, Options{})
	a.NoError(err)
	a.NotNil(g)

	player1.hand = deck.CardsFromString("3s,7s,4h,5h,6h")
	g.trumpCard = deck.CardFromString("8h")

	hand := player1.GetValidMoves(g)
	a.Equal(deck.Hand(deck.CardsFromString("3s,7s,4h,5h,6h")), hand)

	g.winningCardPlayed = &playedCard{card: deck.CardFromString("4s")}
	g.cardsPlayed = []*playedCard{g.winningCardPlayed}
	hand = player1.GetValidMoves(g)
	a.Equal(deck.Hand(deck.CardsFromString("7s")), hand)

	g.winningCardPlayed = &playedCard{card: deck.CardFromString("13d")}
	g.cardsPlayed = []*playedCard{g.winningCardPlayed}
	hand = player1.GetValidMoves(g)
	a.Equal(deck.Hand(deck.CardsFromString("4h,5h,6h")), hand)
}
