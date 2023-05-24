package texasholdem

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"testing"
)

func TestGame__lazyPineapple(t *testing.T) {
	assertLazyPineapple(t, "2c,3c,4c", "14d,14h,5h", "14c,5c,14s,8s,9s", func(p1Adj int, p1Hand string, p2Adj int, p2Hand string) {
		assert.Equal(t, -75, p1Adj)
		assert.Equal(t, 75, p2Adj)
		assert.Equal(t, "Pair", p1Hand)
		assert.Equal(t, "Four of a kind", p2Hand)
	})
}

// ensure that three cards can't be used from the hand to make a straight
func TestGame__lazyPineapple_badStraight(t *testing.T) {
	assertLazyPineapple(t, "2c,3c,4c", "10c,11c,12c", "5d,6d,8d,13d,13s", func(p1Adj int, p1Hand string, p2Adj int, p2Hand string) {
		assert.Equal(t, -75, p1Adj)
		assert.Equal(t, 75, p2Adj)
		assert.Equal(t, "Pair", p1Hand) // must not be a straight
		assert.Equal(t, "Pair", p2Hand)
	})
}

func assertLazyPineapple(t *testing.T, p1, p2, community string, callback func(p1Adj int, p1Hand string, p2Adj int, p2Hand string)) {
	t.Helper()

	a := assert.New(t)

	opts := DefaultOptions()
	opts.Variant = LazyPineapple

	game := setupNewGame(opts, 1000, 1000)
	assertTick(t, game)
	a.Equal(3, len(game.participants[1].cards))

	game.participants[1].cards = deck.CardsFromString(p1)
	game.participants[2].cards = deck.CardsFromString(p2)
	game.deck.Cards = deck.CardsFromString(community)

	assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound)

	{
		assertAction(t, game, 2, action.Call)
		assertAction(t, game, 1, action.Check)

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

		callback(
			details.BalanceAdjustments[1],
			game.participants[1].handAnalyzer.GetHand().String(),
			details.BalanceAdjustments[2],
			game.participants[2].handAnalyzer.GetHand().String(),
		)
	}
}
