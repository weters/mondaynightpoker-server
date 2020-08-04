package sevencard

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestFollowTheQueen_ParticipantReceivedCard(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	opts.Variant = &FollowTheQueen{}
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	a.NoError(err)

	game.deck = deck.New()
	game.deck.SetSeed(0)
	game.deck.Cards = deck.CardsFromString("12c,2c,5c," +
		"3c,4c,6c," +
		"5d,6d,7d")

	a.NoError(game.Start())

	p := createParticipantGetter(game)
	a.Equal("!12c,3c,5d", deck.CardsToString(p(1).hand))
	a.Equal("2c,4c,6d", deck.CardsToString(p(2).hand))
	a.Equal("5c,6c,7d", deck.CardsToString(p(3).hand))

	game.deck.Cards = deck.CardsFromString("2c,12d,3c")

	a.NoError(game.participantChecks(p(3)))
	a.NoError(game.participantChecks(p(1)))
	a.NoError(game.participantChecks(p(2)))

	a.Equal("!12c,!3c,5d,2c", deck.CardsToString(p(1).hand))
	a.Equal("2c,4c,6d,!12d", deck.CardsToString(p(2).hand))
	a.Equal("5c,6c,7d,!3c", deck.CardsToString(p(3).hand))

	game.deck.Cards = deck.CardsFromString("12s,10d,14c")

	a.NoError(game.participantChecks(p(3)))
	a.NoError(game.participantChecks(p(1)))
	a.NoError(game.participantChecks(p(2)))

	a.Equal("!12c,3c,5d,2c,!12s", deck.CardsToString(p(1).hand))
	a.Equal("2c,4c,6d,!12d,!10d", deck.CardsToString(p(2).hand))
	a.Equal("5c,6c,7d,3c,14c", deck.CardsToString(p(3).hand))
}
