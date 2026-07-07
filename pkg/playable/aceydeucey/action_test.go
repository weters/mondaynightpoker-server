package aceydeucey

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestAction_String(t *testing.T) {
	a := assert.New(t)
	a.Equal(ActionPickAceLow.String(), "Pick Low Ace")
	a.Equal(ActionPickAceHigh.String(), "Pick High Ace")
	a.Equal(ActionBet.String(), "Bet")
	a.Equal(ActionBetTheGap.String(), "Bet the Gap")
	a.Equal(ActionPass.String(), "Pass")
	a.Equal(ActionPending.String(), "Pending")

	a.PanicsWithValue("invalid action: -1", func() {
		_ = Action(-1).String()
	})
}

func TestAction_ID(t *testing.T) {
	a := assert.New(t)
	a.Equal("pending", ActionPending.ID())
	a.Equal("pick-ace-low", ActionPickAceLow.ID())
	a.Equal("pick-ace-high", ActionPickAceHigh.ID())
	a.Equal("bet", ActionBet.ID())
	a.Equal("bet-the-gap", ActionBetTheGap.ID())
	a.Equal("pass", ActionPass.ID())

	a.PanicsWithValue("invalid action: -1", func() {
		_ = Action(-1).ID()
	})
}

func TestActionFromID(t *testing.T) {
	a := assert.New(t)

	for expected := ActionPending; expected <= ActionPass; expected++ {
		action, err := ActionFromID(expected.ID())
		a.Equal(expected, action)
		a.NoError(err)
	}

	action, err := ActionFromID("abc")
	a.Equal(-1, int(action))
	a.EqualError(err, "invalid action: abc")
}

func TestAction_MarshalJSON(t *testing.T) {
	b, err := json.Marshal(ActionBetTheGap)
	assert.NoError(t, err)
	assert.Equal(t, `{"id":"bet-the-gap","name":"Bet the Gap"}`, string(b))
}

func TestAceyDeucey_getActionsForParticipant(t *testing.T) {
	a := assert.New(t)

	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())
	a.NoError(err)

	game.deck.Cards = deck.CardsFromString("2c,5c,3c")
	a.Nil(game.getActionsForParticipant(1))

	a.NoError(game.getCurrentRound().DealCard())
	a.Nil(game.getActionsForParticipant(1))

	a.NoError(game.getCurrentRound().DealCard())
	a.Equal([]Action{ActionBet}, game.getActionsForParticipant(1))
	game.options.AllowPass = true
	a.Equal([]Action{ActionPass, ActionBet}, game.getActionsForParticipant(1))
	game.options.AllowPass = false

	a.Nil(game.getActionsForParticipant(2))
	a.Nil(game.getActionsForParticipant(3))

	game.getCurrentRound().Games[0].LastCard = deck.CardFromString("4c")
	a.Equal([]Action{ActionBet}, game.getActionsForParticipant(1))
	game.getCurrentRound().Pot = betTheGapAmount * 2
	a.Equal([]Action{ActionBet, ActionBetTheGap}, game.getActionsForParticipant(1))

	game.options.AllowPass = true
	a.Equal([]Action{ActionPass, ActionBet, ActionBetTheGap}, game.getActionsForParticipant(1))

	// test ace
	game, err = NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())
	a.NoError(err)

	game.deck.Cards = deck.CardsFromString("14s,5c,3c")
	a.Nil(game.getActionsForParticipant(1))

	a.NoError(game.getCurrentRound().DealCard())
	a.Equal([]Action{ActionPickAceLow, ActionPickAceHigh}, game.getActionsForParticipant(1))
	a.Nil(game.getActionsForParticipant(2))
	a.Nil(game.getActionsForParticipant(3))
}
