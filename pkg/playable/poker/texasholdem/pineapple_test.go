package texasholdem

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"testing"
)

func TestGame__lazyPineapple(t *testing.T) {
	a := assert.New(t)

	opts := DefaultOptions()
	opts.Variant = LazyPineapple

	game := setupNewGame(opts, 1000, 1000)
	assertTick(t, game)
	a.Equal(3, len(game.participants[1].cards))

	game.participants[1].cards = deck.CardsFromString("2c,3c,4c")
	game.participants[2].cards = deck.CardsFromString("14d,14h,5h")
	game.deck.Cards = deck.CardsFromString("14c,5c,14s,8s,9s")

	assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound)

	{
		assertAction(t, game, 1, action.Call)
		assertAction(t, game, 2, action.Check)

		assertTickFromWaiting(t, game, DealerStateDealFlop)
	}

	{
		assertTick(t, game)
		assertAction(t, game, 1, action.Check)
		assertAction(t, game, 2, action.Check)

		assertTickFromWaiting(t, game, DealerStateDealTurn)
	}

	{
		assertTick(t, game)
		assertAction(t, game, 1, action.Check)
		assertAction(t, game, 2, action.Check)

		assertTickFromWaiting(t, game, DealerStateDealRiver)
	}

	{
		assertTick(t, game)
		assertAction(t, game, 1, action.Check)
		assertAction(t, game, 2, action.Check)

		assertTickFromWaiting(t, game, DealerStateRevealWinner)
	}

	{
		assertTick(t, game)
		assertTickFromWaiting(t, game, DealerStateEnd)
		assertTick(t, game)
		details, ok := game.GetEndOfGameDetails()
		a.True(ok)
		a.Equal(map[int64]int{
			1: 75,
			2: -75,
		}, details.BalanceAdjustments)
	}
}
