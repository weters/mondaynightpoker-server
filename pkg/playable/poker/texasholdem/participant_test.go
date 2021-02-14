package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
