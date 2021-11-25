package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"mondaynightpoker-server/pkg/snapshot"
	"testing"
)

func TestGame_Name(t *testing.T) {
	p := setupParticipants(1000, 1000)
	g, err := NewGame(logrus.StandardLogger(), p, DefaultOptions())
	assert.NoError(t, err)
	assert.Equal(t, "Texas Hold'em (${25}/${50})", g.Name())

	g, err = NewGame(logrus.StandardLogger(), p, Options{
		Variant:    Standard,
		SmallBlind: 50,
		BigBlind:   100,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Texas Hold'em (${50}/${100})", g.Name())

	g, err = NewGame(logrus.StandardLogger(), p, Options{Variant: Standard, Ante: -1})
	assert.EqualError(t, err, "ante must be at least ${0}")
	assert.Nil(t, g)
}

func TestGame_Key(t *testing.T) {
	assert.Equal(t, "texas-hold-em", (&Game{}).Key())
}

func TestNameFromOptions(t *testing.T) {
	assert.Equal(t, "", NameFromOptions(Options{Ante: -1}))

	opts := DefaultOptions()
	assert.Equal(t, "Texas Hold'em (${25}/${50})", NameFromOptions(opts))

	opts.Variant = Pineapple
	assert.Equal(t, "Pineapple (${25}/${50})", NameFromOptions(opts))

	opts.Variant = LazyPineapple
	opts.SmallBlind = 75
	opts.BigBlind = 125
	assert.Equal(t, "Lazy Pineapple (${75}/${125})", NameFromOptions(opts))
}

func TestGame_GetPlayerState_nonParticipantID(t *testing.T) {
	a := assert.New(t)

	game := setupNewGame(DefaultOptions(), 1000, 1000, 1000)

	game.deck = deck.New() // provide a consistent deck for testing purposes

	assertTick(t, game)
	assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound)

	a.Equal(2, len(game.participants[1].cards), "ensure cards have been dealt to players")

	ps, err := game.GetPlayerState(4)
	a.NoError(err)
	a.NotNil(ps)
	snapshot.ValidateSnapshot(t, ps, 0, "ensure game state returned for player not in game")
}

func TestGame_validateBetOrRaise(t *testing.T) {
	a := assert.New(t)

	newGame := func(t *testing.T, ante, smallBlind, bigBlind, tableStake int) *Game {
		t.Helper()

		game := setupNewGame(Options{Standard, ante, smallBlind, bigBlind}, tableStake, tableStake)
		assertTick(t, game)
		assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound)

		assertAction(t, game, 1, game.ActionsForParticipant(1)[0])
		assertAction(t, game, 2, action.Check)

		assertTickFromWaiting(t, game, DealerStateDealFlop)
		assertTick(t, game)

		return game
	}

	// bet
	{
		game := newGame(t, 0, 0, 0, 1000)
		a.EqualError(game.validateBetOrRaise(game.participants[1], 24), "bet must be in increments of ${25}")
		a.EqualError(game.validateBetOrRaise(game.participants[1], 0), "bet must be at least ${25}")

		game = newGame(t, 50, 100, 150, 1000)
		a.EqualError(game.validateBetOrRaise(game.participants[1], 125), "bet must be at least ${150}")
		a.EqualError(game.validateBetOrRaise(game.participants[1], 425), "bet must be at most ${400}")

		game.participants[1].tableStake = 325
		a.EqualError(game.validateBetOrRaise(game.participants[1], 100), "bet must be at least ${150}")
		a.NoError(game.validateBetOrRaise(game.participants[1], 125), "allow all-in")
	}

	// raise
	{
		game := newGame(t, 50, 100, 150, 1000)
		assertActionAndAmount(t, game, 1, action.Bet, 150)

		a.EqualError(game.validateBetOrRaise(game.participants[2], 275), "raise must be to at least ${300}")
		a.EqualError(game.validateBetOrRaise(game.participants[2], 875), "raise must not exceed total of ${850}")
		a.EqualError(game.validateBetOrRaise(game.participants[2], 125), "you cannot raise to an amount less than the current bet")

		game.participants[2].tableStake = 475
		a.EqualError(game.validateBetOrRaise(game.participants[2], 250), "raise must be to at least ${300}")
		a.NoError(game.validateBetOrRaise(game.participants[2], 275), "allow all-in")
	}
}
