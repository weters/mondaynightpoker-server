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

func TestParticipant_participantJSON(t *testing.T) {
	game := &Game{community: make(deck.Hand, 0)}
	p := &Participant{
		cards:  deck.CardsFromString("2c,3c"),
		reveal: false,
	}

	record := p.participantJSON(game, true)
	assert.NotNil(t, record.Cards)

	record = p.participantJSON(game, false)
	assert.Nil(t, record.Cards)

	p.reveal = true
	record = p.participantJSON(game, false)
	assert.NotNil(t, record.Cards)
}

func TestGame_participantIsPendingTurn(t *testing.T) {
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4, 5}, DefaultOptions())
	game.newRoundSetup()

	a := assert.New(t)
	a.False(game.isParticipantPendingTurn(3), "not in betting round")

	game.dealerState = DealerStateFinalBettingRound
	a.False(game.isParticipantPendingTurn(1))
	a.True(game.isParticipantPendingTurn(2))
	a.True(game.isParticipantPendingTurn(3))
	a.True(game.isParticipantPendingTurn(4))
	a.True(game.isParticipantPendingTurn(5))

	game.decisionIndex = 4
	a.False(game.isParticipantPendingTurn(1))
	a.False(game.isParticipantPendingTurn(2))
	a.False(game.isParticipantPendingTurn(3))
	a.False(game.isParticipantPendingTurn(4))
	a.False(game.isParticipantPendingTurn(5))

	game.decisionIndex = 1
	game.decisionStart = 3
	a.True(game.isParticipantPendingTurn(1))
	a.True(game.isParticipantPendingTurn(2))
	a.True(game.isParticipantPendingTurn(3))
	a.False(game.isParticipantPendingTurn(4))
	a.False(game.isParticipantPendingTurn(5))
}

func TestGame_FutureActionsForParticipant(t *testing.T) {
	a := assert.New(t)

	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())
	a.NotNil(game)

	assertTick(t, game, "move into dealing cards")
	assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound, "put in betting round")
	a.Equal(DealerStatePreFlopBettingRound, game.dealerState)

	a.Equal([]Action{{"Call", 50}, {"Raise", 200}, actionFold}, game.FutureActionsForParticipant(1))
	a.Equal([]Action{actionCheck, {"Raise", 200}, actionFold}, game.FutureActionsForParticipant(2))
	a.Nil(game.FutureActionsForParticipant(3), "player three has current actions")

	game.newRoundSetup()
	game.dealerState = DealerStateTurnBettingRound

	a.Nil(game.FutureActionsForParticipant(1), "player one has current actions")
	a.Equal([]Action{actionCheck, {"Bet", 200}, actionFold}, game.FutureActionsForParticipant(2))
	a.Equal([]Action{actionCheck, {"Bet", 200}, actionFold}, game.FutureActionsForParticipant(3))

	game.decisionIndex = 1
	game.currentBet = 600
	a.Nil(game.FutureActionsForParticipant(1), "player one already went")
	a.Nil(game.FutureActionsForParticipant(2), "player two has current actions")
	game.participants[3].bet = 200
	a.Equal([]Action{{"Call", 400}, {"Raise", 800}, actionFold}, game.FutureActionsForParticipant(3))

	game.decisionStart = 2
	game.decisionIndex = 1
	game.currentBet = 800
	a.Nil(game.FutureActionsForParticipant(1), "player one has current actions")
	game.participants[2].bet = 600
	a.Equal([]Action{{"Call", 200}, actionFold}, game.FutureActionsForParticipant(2))
	a.Nil(game.FutureActionsForParticipant(3), "player three already went")

	game.decisionStart = 0
	game.decisionIndex = 0
	game.currentBet = 200
	game.participants[2].bet = 200
}
