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
	r := NewRound(DefaultOptions(), 1, d, 50)

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
			a.EqualError(r.DealCard(), "cannot deal card from state: pending-ace-decision")
			a.Equal(51, r.deck.CardsLeft())

			a.NoError(r.SetAce(aceHigh[0]))
			a.EqualError(r.SetAce(aceHigh[0]), "cannot choose ace low/high from state: first-card-dealt")
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

func TestRound_pass(t *testing.T) {
	a := assert.New(t)
	r := createTestRound(500, "2c,4c")
	a.NoError(r.DealCard())

	a.EqualError(r.SetPass(), "cannot pass from state: first-card-dealt")

	a.NoError(r.DealCard())
	a.NoError(r.SetPass())
	a.Equal(RoundStatePassed, r.State)
	r.PassRound()
	simulateWait(r)
	a.Equal(RoundStateRoundOver, r.State)

	a.Equal(0, r.ParticipantAdjustments())
	a.Equal(500, r.Pot)
}

func TestRound_betTheGap(t *testing.T) {
	a := assert.New(t)
	r := createTestRound(1025, "2c,4c,3c")
	a.NoError(r.DealCard())
	a.NoError(r.DealCard())
	a.NoError(r.SetBet(50, true))
	a.NoError(r.DealCard())
	simulateWait(r)
	a.Equal(500, r.ParticipantAdjustments())
	a.Equal(525, r.Pot)

	// no win
	r = createTestRound(1025, "2c,4c,5c")
	a.NoError(r.DealCard())
	a.NoError(r.DealCard())
	a.NoError(r.SetBet(50, true))
	a.NoError(r.DealCard())
	simulateWait(r)
	a.Equal(-50, r.ParticipantAdjustments())
	a.Equal(1075, r.Pot)
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

func TestRound_SetBet(t *testing.T) {
	a := assert.New(t)
	r := createTestRound(125, "3c,14c")
	a.NoError(r.DealCard())

	a.EqualError(r.SetBet(25, false), "cannot place a bet from state: first-card-dealt")

	a.NoError(r.DealCard())

	// don't allow a bet if the game
	r.Games[0].gameOver = true
	a.EqualError(r.SetBet(25, false), "game is over")

	r.Games[0].gameOver = false
	a.EqualError(r.SetBet(26, false), "bet must be in increments of ${25}")
	a.EqualError(r.SetBet(150, false), "bet of ${150} exceeds the max bet of ${125}")
	a.EqualError(r.SetBet(0, false), "bet must be at least ${25}")
	a.EqualError(r.SetBet(50, true), "bet the gap for half-pot requires a one-card gap")
	a.Equal(RoundStatePendingBet, r.State)
	a.NoError(r.SetBet(25, false))
	a.Equal(RoundStateBetPlaced, r.State)

	r.State = RoundStatePendingBet
	a.NoError(r.SetBet(125, false))

	assertTestTheGapSuccessful := func(t *testing.T, cards string, highAce ...bool) {
		t.Helper()
		a := assert.New(t)

		r := createTestRound(100, cards)
		a.NoError(r.DealCard())

		if len(highAce) > 0 {
			a.NoError(r.SetAce(highAce[0]))
		}

		a.NoError(r.DealCard())
		a.NoError(r.SetBet(25, true))
	}

	assertTestTheGapSuccessful(t, "2c,4c")
	assertTestTheGapSuccessful(t, "4c,2c")
	assertTestTheGapSuccessful(t, "12c,14c")
	assertTestTheGapSuccessful(t, "14c,3c", false)

	// ensure you can't bet the gap when there's a quarter in the pot
	r = createTestRound(25, "2c,4c")
	a.NoError(r.DealCard())
	a.NoError(r.DealCard())
	a.EqualError(r.SetBet(50, true), "bet of ${50} exceeds the max bet of ${25}")
}

func TestRound_canBetTheGap(t *testing.T) {
	a := assert.New(t)

	r := createTestRound(100, "")
	r.Games[0].FirstCard = deck.CardFromString("2c")
	r.Games[0].LastCard = deck.CardFromString("4c")
	r.State = RoundStatePendingBet
	a.True(r.canBetTheGap())

	r.Pot = 75
	a.False(r.canBetTheGap())
	r.Pot = 100

	r.State = RoundStateFirstCardDealt
	assert.False(t, r.canBetTheGap())

	// back to a good state
	r.State = RoundStatePendingBet

	r.Games[0].gameOver = true
	assert.False(t, r.canBetTheGap())
}

func TestRound_drawCard(t *testing.T) {
	r := createTestRound(100, "")
	r.Games = []*SingleGame{
		{
			FirstCard:  deck.CardFromString("2d"),
			MiddleCard: deck.CardFromString("3d"),
			LastCard:   deck.CardFromString("4d"),
			gameOver:   true,
		},
		{
			FirstCard: deck.CardFromString("2h"),
			LastCard:  deck.CardFromString("4h"),
		},
	}
	r.deck.Cards = deck.CardsFromString("2c,3c")

	assertDrawCard := func(r *Round, expectedCard string) {
		t.Helper()

		card, err := r.drawCard()
		assert.NoError(t, err)
		assert.Equal(t, expectedCard, deck.CardToString(card))
	}

	assertDrawCard(r, "2c")
	assertDrawCard(r, "3c")
	assert.Equal(t, 0, r.deck.CardsLeft())
	r.deck.SetSeed(1)
	assertDrawCard(r, "14c")
	assert.Equal(t, 49, r.deck.CardsLeft())

	foundCard := make(map[string]bool)
	for _, card := range r.deck.Cards {
		foundCard[deck.CardToString(card)] = true
	}

	// only the cards from the unfinished game aren't shuffled in
	assert.True(t, foundCard["2d"])
	assert.True(t, foundCard["3d"])
	assert.True(t, foundCard["4d"])
	assert.True(t, foundCard["2c"])
	assert.True(t, foundCard["3c"])
	assert.False(t, foundCard["2h"])
	assert.False(t, foundCard["4h"])
	assert.False(t, foundCard["14c"])
}

func TestRound_drawCardWithChaos(t *testing.T) {
	opts := DefaultOptions()
	opts.GameType = GameTypeStandard
	d := deck.New()

	assertDrawCard := func(card *deck.Card, err error) {
		t.Helper()
		assert.NoError(t, err)
		assert.NotNil(t, card)
	}

	r := NewRound(opts, 0, d, 100)
	assertDrawCard(r.drawCard())
	assertDrawCard(r.drawCard())
	assertDrawCard(r.drawCard())
	assert.Equal(t, 49, d.CardsLeft())

	opts.GameType = GameTypeChaos
	r = NewRound(opts, 0, d, 100)
	assertDrawCard(r.drawCard())
	assertDrawCard(r.drawCard())
	assertDrawCard(r.drawCard())
	assert.Equal(t, 51, d.CardsLeft())
}

func TestRound_getActiveCardsInGame(t *testing.T) {
	a := assert.New(t)
	r := createTestRound(125, "")
	r.Games = []*SingleGame{
		{
			FirstCard:  deck.CardFromString("2c"),
			MiddleCard: deck.CardFromString("3c"),
			LastCard:   deck.CardFromString("4c"),
			gameOver:   true,
		},
		{
			FirstCard: deck.CardFromString("2d"),
			LastCard:  deck.CardFromString("3d"),
		},
	}

	r.State = RoundStateRoundOver
	a.Nil(r.getCardsInActiveGame())

	r.State = RoundStateBetPlaced
	a.Equal("2d,3d", deck.CardsToString(r.getCardsInActiveGame()))

	r.Games = []*SingleGame{
		{
			FirstCard:  deck.CardFromString("2c"),
			MiddleCard: deck.CardFromString("3c"),
			LastCard:   deck.CardFromString("4c"),
			gameOver:   true,
		},
		{},
	}

	a.Nil(r.getCardsInActiveGame())
}

func TestRound_SetAce(t *testing.T) {
	a := assert.New(t)
	r := createTestRound(125, "14c")
	a.NoError(r.DealCard())

	a.True(r.Games[0].FirstCard.IsBitSet(aceStateUndecided))
	a.NoError(r.SetAce(false))
	a.False(r.Games[0].FirstCard.IsBitSet(aceStateUndecided))
	a.False(r.Games[0].FirstCard.IsBitSet(aceStateHigh))
	a.True(r.Games[0].FirstCard.IsBitSet(aceStateLow))

	a.EqualError(r.SetAce(false), "cannot choose ace low/high from state: first-card-dealt")

	r.State = RoundStatePendingAceDecision
	a.NoError(r.SetAce(true))
	a.False(r.Games[0].FirstCard.IsBitSet(aceStateUndecided))
	a.True(r.Games[0].FirstCard.IsBitSet(aceStateHigh))
	a.False(r.Games[0].FirstCard.IsBitSet(aceStateLow))

	// this should NEVER happen
	r = createTestRound(125, "5c")
	a.NoError(r.DealCard())
	r.State = RoundStatePendingAceDecision
	a.PanicsWithValue("first card is 5♣, but the state is pending-ace-decision", func() { _ = r.SetAce(false) })
}

func TestRound_nextGameState(t *testing.T) {
	a := assert.New(t)

	r := createTestRound(125, "")
	r.Games = []*SingleGame{
		{gameOver: true},
		{FirstCard: deck.CardFromString("5s")},
	}
	r.State = RoundStateGameOver
	a.NoError(r.nextGame())
	a.Equal(RoundStateFirstCardDealt, r.State)

	r = createTestRound(125, "")
	r.Games = []*SingleGame{
		{gameOver: true},
		{FirstCard: deck.CardFromString("14s")},
	}
	r.State = RoundStateGameOver
	a.NoError(r.nextGame())
	a.Equal(RoundStatePendingAceDecision, r.State)
}

func TestRound_panicWithLastCardWhenAceNotSet(t *testing.T) {
	a := assert.New(t)
	r := createTestRound(125, "14c,2c,4c")
	a.NoError(r.DealCard())
	r.State = RoundStateFirstCardDealt // this should NEVER happen

	a.PanicsWithValue("bit not properly set on first ace", func() {
		_ = r.DealCard()
	})
}

func createTestRound(pot int, cards string) *Round {
	d := deck.New()
	for i, card := range deck.CardsFromString(cards) {
		d.Cards[i] = card
	}

	return NewRound(DefaultOptions(), 1, d, pot)
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

func TestRound_halfPot(t *testing.T) {
	a := assert.New(t)
	r := createTestRound(100, "")

	a.Equal(50, r.getHalfPot())

	r.Pot = 75
	a.Equal(25, r.getHalfPot())

	r.Pot = 125
	a.Equal(50, r.getHalfPot())
}

func TestRound_getMaxBet(t *testing.T) {
	a := assert.New(t)
	r := &Round{Pot: 200}

	// test get full pot
	a.Equal(200, r.getMaxBet())

	testHalfPot := func(t *testing.T, pot, expects int) {
		t.Helper()

		r.HalfPotMax = true
		r.Pot = pot
		a.Equal(expects, r.getMaxBet())
	}

	testHalfPot(t, 225, 100)
	testHalfPot(t, 200, 100)
	testHalfPot(t, 175, 75)
	testHalfPot(t, 150, 75)
	testHalfPot(t, 50, 25)
	testHalfPot(t, 25, 25)
	testHalfPot(t, 0, 0)
}

func TestRound_Pass(t *testing.T) {
	a := assert.New(t)
	r := &Round{}
	r.State = RoundStateStart
	a.EqualError(r.SetPass(), "cannot pass from state: start")

	r.State = RoundStatePendingBet
	a.NoError(r.SetPass())
	a.Equal(RoundStatePassed, r.State)
}
