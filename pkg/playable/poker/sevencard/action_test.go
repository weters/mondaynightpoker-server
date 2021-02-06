package sevencard

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAction_String(t *testing.T) {
	a := assert.New(t)

	a.Equal("Fold", ActionFold.String())
	a.Equal("Raise", ActionRaise.String())

	a.PanicsWithValue("unknown action: bad", func() {
		_ = Action("bad").String()
	})
}

func TestGame_getActionsForParticipant(t *testing.T) {
	a := assert.New(t)
	game, p := createTestGame()

	a.Equal([]Action{ActionFold, ActionCheck, ActionBet}, game.getActionsForParticipant(p(1)))
	a.Equal([]Action{}, game.getActionsForParticipant(p(2)))
	a.Equal([]Action{}, game.getActionsForParticipant(p(3)))
	a.NoError(game.participantChecks(p(1)))

	a.Equal([]Action{ActionFold, ActionCheck, ActionBet}, game.getActionsForParticipant(p(2)))
	a.Equal([]Action{}, game.getActionsForParticipant(p(1)))
	a.Equal([]Action{}, game.getActionsForParticipant(p(3)))
	a.NoError(game.participantBets(p(2), 25))

	a.Equal([]Action{ActionFold, ActionCall, ActionRaise}, game.getActionsForParticipant(p(3)))
	a.Equal([]Action{}, game.getActionsForParticipant(p(1)))
	a.Equal([]Action{}, game.getActionsForParticipant(p(2)))

	game.endGame()

	// participants no longer have to end the game manually. ensure there
	// are no actions for them
	a.Equal([]Action{}, game.getActionsForParticipant(p(1)))
	a.Equal([]Action{}, game.getActionsForParticipant(p(2)))
	a.Equal([]Action{}, game.getActionsForParticipant(p(3)))
}

func TestAction_MarshalJSON(t *testing.T) {
	b, err := ActionBet.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"id":"bet","name":"Bet"}`, string(b))
}

func TestActionFromString(t *testing.T) {
	a := assert.New(t)

	action, err := ActionFromString("fold")
	a.NoError(err)
	a.Equal(ActionFold, action)

	action, err = ActionFromString("bad-action")
	a.EqualError(err, "unknown action: bad-action")
	a.Equal("", string(action))
}
