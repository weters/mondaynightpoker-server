package aceydeucey

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestSingleGame_firstCardRank(t *testing.T) {
	a := assert.New(t)

	game := &SingleGame{}
	a.PanicsWithValue("FirstCard is not set", func() {
		_ = game.firstCardRank()
	})

	game.FirstCard = deck.CardFromString("14s")
	a.Equal(deck.HighAce, game.firstCardRank())

	game.FirstCard.SetBit(aceStateLow)
	a.Equal(deck.LowAce, game.firstCardRank())

	game.FirstCard = deck.CardFromString("5c")
	a.Equal(5, game.firstCardRank())
}
