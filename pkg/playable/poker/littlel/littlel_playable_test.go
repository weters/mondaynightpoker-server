package littlel

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestGame_getMinBet(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	opts.Ante = 75
	game, err := newGame(opts, 1000, 1000)
	a.NoError(err)
	a.Equal(75, game.getMinBet())

	a.NoError(game.tradeCardsForParticipant(game.idToParticipant[1], []*deck.Card{}))
	a.NoError(game.tradeCardsForParticipant(game.idToParticipant[2], []*deck.Card{}))
	a.NoError(game.NextRound())

	a.NoError(game.ParticipantBets(game.idToParticipant[1], 75))
	a.NoError(game.ParticipantBets(game.idToParticipant[2], 200))
	a.Equal(325, game.getMinBet())
}
