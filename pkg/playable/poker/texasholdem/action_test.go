package texasholdem

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAction(t *testing.T) {
	assertAction := func(t *testing.T, id int, action Action, name string) {
		t.Helper()

		a, err := ActionFromInt(id)
		assert.NoError(t, err)
		assert.Equal(t, action, a)
		assert.Equal(t, name, a.String())
	}

	assertAction(t, 0, ActionCheck, "Check")
	assertAction(t, 1, ActionCall, "Call")
	assertAction(t, 2, ActionBet, "Bet")
	assertAction(t, 3, ActionRaise, "Raise")
	assertAction(t, 4, ActionFold, "Fold")

	_, err := ActionFromInt(-1)
	assert.EqualError(t, err, "no action with id -1")

	_, err = ActionFromInt(5)
	assert.EqualError(t, err, "no action with id 5")
}
