package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGame_ActionsForParticipant(t *testing.T) {
	opts := Options{
		LowerLimit: 100,
		UpperLimit: 200,
	}

	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)

	a := assert.New(t)
	a.Nil(game.ActionsForParticipant(1))
	a.Nil(game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.dealerState = DealerStateFlopBettingRound
	a.Equal([]Action{ActionCheck, ActionBet, ActionFold}, game.ActionsForParticipant(1))
	a.Nil(game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.decisionIndex = 1
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{ActionCheck, ActionBet, ActionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.currentBet = 100
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{ActionCall, ActionRaise, ActionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.participants[2].bet = 100
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{ActionCheck, ActionRaise, ActionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.currentBet = 400
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{ActionCall, ActionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))

	game.participants[2].bet = 400
	a.Nil(game.ActionsForParticipant(1))
	a.Equal([]Action{ActionCheck, ActionFold}, game.ActionsForParticipant(2))
	a.Nil(game.ActionsForParticipant(3))
}
