package aceydeucey

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
	"time"
)

func TestNewRound(t *testing.T) {
	a := assert.New(t)
	d := deck.New()
	r := NewRound(d, 50)

	a.Equal(50, r.Pot)
	a.Equal(RoundStateStart, r.State)
	a.Equal(0, r.activeGameIndex)
	a.Equal(1, len(r.Games))
}

func TestRound_standardGames(t *testing.T) {
	test := func(t *testing.T, cards string, pot, bet int, result SingleGameResult, adjustment int, aceHigh ...bool) {
		t.Helper()

		a := assert.New(t)
		r := createTestRound(100, cards)

		// deal first card
		a.NoError(r.DealCard())

		if len(aceHigh) > 0 {
			a.EqualError(r.DealCard(), "ace has not been decided")
			a.Equal(51, r.deck.CardsLeft())

			a.NoError(r.SetAce(aceHigh[0]))
			a.EqualError(r.SetAce(aceHigh[0]), "ace has already been decided")
		}

		a.Equal(RoundStateFirstCardDealt, r.State)

		// deal second card
		a.NoError(r.DealCard())
		a.Equal(RoundStatePendingBet, r.State)

		// verify state
		a.Equal(50, r.deck.CardsLeft())

		// ensure you can't deal without betting
		a.EqualError(r.DealCard(), "cannot deal card from state: pending-bet")
		a.Equal(50, r.deck.CardsLeft()) // ensure same number of cards in deck

		// place bet
		a.NoError(r.SetBet(bet, false))
		a.Equal(RoundStateBetPlaced, r.State)

		// final deal
		a.NoError(r.DealCard())
		a.EqualError(r.DealCard(), "game is over")

		a.Equal(RoundStateWaiting, r.State)
		simulateWait(r)
		a.Equal(RoundStateRoundOver, r.State)

		a.Equal(result, r.Games[0].Result)
		a.Equal(adjustment, r.Games[0].Adjustment)
		a.Equal(pot-adjustment, r.Pot)
	}

	// test some standard win scenarios
	test(t, "2c,4c,3c", 100, 25, SingleGameResultWon, 25)          // 2 3 4
	test(t, "4c,2c,3d", 100, 25, SingleGameResultWon, 25)          // 4 3 2
	test(t, "12c,14c,13d", 100, 25, SingleGameResultWon, 25)       // Q K A
	test(t, "14c,12c,13d", 100, 25, SingleGameResultWon, 25, true) // A K Q (ace-high)
	test(t, "14c,13d,5c", 100, 25, SingleGameResultWon, 25, false) // A 5 K (ace-low)

	// test some standard loss scenarios
	test(t, "3c,5c,2c", 100, 25, SingleGameResultLost, -25)          // 3 2 5
	test(t, "3c,5c,6c", 100, 25, SingleGameResultLost, -25)          // 3 2 5
	test(t, "14c,3c,4c", 100, 25, SingleGameResultLost, -25, false)  // A 4 3 (ace-low)
	test(t, "14c,8c,4c", 100, 25, SingleGameResultLost, -25, true)   // A 4 8 (ace-high)
	test(t, "14c,8c,14d", 100, 25, SingleGameResultLost, -25, false) // A A 8 (ace-low)
	test(t, "14c,8c,14d", 100, 25, SingleGameResultPost, -50, true)  // A A 8 (ace-high; post)
	test(t, "4c,6c,4c", 100, 25, SingleGameResultPost, -50)          // 4 4 6 (post)
}

func TestRound_freeGames(t *testing.T) {
	test := func(t *testing.T, cards string, aceHigh ...bool) {
		t.Helper()
		a := assert.New(t)

		r := createTestRound(100, cards)
		a.NoError(r.DealCard())

		if len(aceHigh) > 0 {
			a.NoError(r.SetAce(aceHigh[0]))
		}

		a.NoError(r.DealCard())
		a.EqualError(r.DealCard(), "game is over")

		a.Equal(RoundStateWaiting, r.State)
		a.Equal(RoundStateRoundOver, r.nextAction.NextState)
		a.Equal(SingleGameResultFreeGame, r.Games[0].Result)
		a.Equal(0, r.Games[0].Adjustment)
		a.Equal(100, r.Pot)
	}

	test(t, "2c,3c")
	test(t, "3c,2c")
	test(t, "14c,2c", false)
	test(t, "14c,13c", true)
}

func TestRound_bonusGame(t *testing.T) {
	// will become
	// game 1: 4c 5c 6c
	// game 2: 4d 5d 6d
	// game 3: 4h 5h 6h
	cards := "4c,4d,6c,5c,4h,6d,5d,6h,5h"

	a := assert.New(t)

	c := deck.CardsFromString(cards)

	r := createTestRound(100, cards)
	a.NoError(r.DealCard())
	a.NoError(r.DealCard())
	a.Equal(RoundStateFirstCardDealt, r.State)

	a.NoError(r.DealCard())
	a.Equal(RoundStatePendingBet, r.State)

	a.Equal(2, len(r.Games))
	a.Equal(0, r.activeGameIndex)
	a.Equal(cardsFromArray(c, 0, -1, 2), cardsFromGame(r.Games[0]))
	a.Equal(cardsFromArray(c, 1, -1, -1), cardsFromGame(r.Games[1]))

	a.EqualError(r.nextGame(), "invalid state to move to next game: pending-bet")
	a.NoError(r.SetBet(25, false))
	a.Equal(RoundStateBetPlaced, r.State)
	a.NoError(r.DealCard())

	a.Equal(SingleGameResultWon, r.Games[0].Result)
	a.Equal(SingleGameResult(""), r.Games[1].Result)
	a.Equal(RoundStateWaiting, r.State)
	a.Equal(RoundStateGameOver, r.nextAction.NextState)

	a.EqualError(r.nextGame(), "invalid state to move to next game: waiting")

	simulateWait(r)
	a.NoError(r.nextGame())
	a.Equal(2, len(r.Games))
	a.Equal(1, r.activeGameIndex)

	a.Equal(RoundStateFirstCardDealt, r.State)
	a.NoError(r.DealCard())
	a.Equal(3, len(r.Games)) // new game created
	a.Equal(1, r.activeGameIndex)

	a.Equal(cardsFromArray(c, 0, 3, 2), cardsFromGame(r.Games[0]))
	a.Equal(cardsFromArray(c, 1, -1, -1), cardsFromGame(r.Games[1]))
	a.Equal(cardsFromArray(c, 4, -1, -1), cardsFromGame(r.Games[2]))

	a.NoError(r.DealCard())
	a.Equal(RoundStatePendingBet, r.State)
	a.NoError(r.SetBet(25, false))
	a.NoError(r.DealCard())

	a.Equal(RoundStateWaiting, r.State)
	a.Equal(RoundStateGameOver, r.nextAction.NextState)
	a.Equal(SingleGameResultWon, r.Games[0].Result)
	a.Equal(SingleGameResultWon, r.Games[1].Result)
	a.Equal(SingleGameResult(""), r.Games[2].Result)

	simulateWait(r)
	a.NoError(r.nextGame())
	a.Equal(3, len(r.Games))
	a.Equal(2, r.activeGameIndex)
	a.Equal(RoundStateFirstCardDealt, r.State)

	a.NoError(r.DealCard())
	a.Equal(RoundStatePendingBet, r.State)
	a.NoError(r.SetBet(25, false))
	a.Equal(RoundStateBetPlaced, r.State)

	a.NoError(r.DealCard())
	a.Equal(RoundStateWaiting, r.State)
	a.Equal(RoundStateRoundOver, r.nextAction.NextState)
	simulateWait(r)
	a.EqualError(r.nextGame(), "invalid state to move to next game: round-over")

	a.Equal(25, r.Pot)
	a.Equal(75, r.ParticipantAdjustments())
}

func TestRound_bonusGameWithAce(t *testing.T) {
	a := assert.New(t)

	// two games created with ace-high, ace
	r := createTestRound(125, "14c,14s")
	a.NoError(r.DealCard())
	a.NoError(r.SetAce(true))
	a.NoError(r.DealCard())
	a.Equal(2, len(r.Games))

	// only one game with ace-low, ace
	r = createTestRound(125, "14c,14s")
	a.NoError(r.DealCard())
	a.NoError(r.SetAce(false))
	a.NoError(r.DealCard())
	a.Equal(1, len(r.Games))
}

func createTestRound(pot int, cards string) *Round {
	d := deck.New()
	for i, card := range deck.CardsFromString(cards) {
		d.Cards[i] = card
	}

	return NewRound(d, pot)
}

func cardsFromArray(c []*deck.Card, indexes ...int) string {
	cards := make([]*deck.Card, len(indexes))
	for i, index := range indexes {
		if index < 0 {
			cards[i] = nil
		} else {
			cards[i] = c[index]
		}
	}

	return deck.CardsToString(cards)
}

func cardsFromGame(g *SingleGame) string {
	return deck.CardsToString([]*deck.Card{
		g.FirstCard,
		g.MiddleCard,
		g.LastCard,
	})
}

func simulateWait(r *Round) {
	r.nextAction.Time = time.Time{}
	r.checkWaiting()
}
