package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"math"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
	"testing"
)

func TestGame_ActionsForParticipant(t *testing.T) {
	opts := Options{
		Ante:       25,
		LowerLimit: 100,
		UpperLimit: 200,
	}

	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)

	a := assert.New(t)
	a.Nil(game.ActionsForParticipant(1))
	a.Nil(game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.dealerState = DealerStatePreFlopBettingRound
	a.Nil(game.ActionsForParticipant(1))
	a.Nil(game.ActionsForParticipant(2))
	a.Equal([]Action{{"Call", 100}, {"Raise", 200}, actionFold}, game.ActionsForParticipant(3))

	game.newRoundSetup()
	game.dealerState = DealerStateFlopBettingRound
	a.Equal([]Action{actionCheck, {"Bet", 100}, actionFold}, game.ActionsForParticipant(1))
	a.Nil(game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.decisionIndex = 1
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{actionCheck, {"Bet", 100}, actionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.currentBet = 100
	game.participants[2].bet = 50
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{{"Call", 50}, {"Raise", 200}, actionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.participants[2].bet = 100
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{actionCheck, {"Raise", 200}, actionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.currentBet = 400
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{{"Call", 300}, actionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.participants[2].bet = 400
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{actionCheck, actionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))
}

func TestGame_ActionsForParticipant_panicsInInvalidState(t *testing.T) {
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())

	// we should never be in this state
	game.newRoundSetup()
	game.dealerState = DealerStateTurnBettingRound
	game.decisionIndex = math.MaxInt32

	assert.PanicsWithError(t, "betting round is over", func() {
		game.ActionsForParticipant(1)
	})
}

func TestParticipant_getHandAnalyzer(t *testing.T) {
	a := assert.New(t)
	p := newParticipant(1)
	p.getHandAnalyzer(nil)
	a.Nil(p.getHandAnalyzer(nil))

	p.cards = deck.CardsFromString("2c,2d,2h,2s,3c")
	a.NotNil(p.getHandAnalyzer(nil))
	a.Equal("Four of a kind", p.getHandAnalyzer(nil).GetHand().String())

	// override the hand analyzer to make sure caching still works
	p.handAnalyzer = handanalyzer.New(5, deck.CardsFromString("2c,2d,3c,3d,5c"))
	a.Equal("Two pair", p.getHandAnalyzer(nil).GetHand().String(), "cached value returned")

	a.Equal("Four of a kind", p.getHandAnalyzer(deck.CardsFromString("9d")).GetHand().String(), "cache is busted")
}
