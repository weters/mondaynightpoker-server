package aceydeucey

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestAction_String(t *testing.T) {
	a := assert.New(t)
	a.Equal(ActionPickAceLow.String(), "Pick Low Ace")
	a.Equal(ActionPickAceHigh.String(), "Pick High Ace")
	a.Equal(ActionBet.String(), "Bet")
	a.Equal(ActionPass.String(), "Pass")
	a.Equal(ActionContinue.String(), "Continue")

	a.PanicsWithValue("invalid action: -1", func() {
		_ = Action(-1).String()
	})
}

func TestActionFromString(t *testing.T) {
	a := assert.New(t)

	action, err := ActionFromString("0")
	a.Equal(ActionPending, action)
	a.NoError(err)

	action, err = ActionFromString("-1")
	a.Equal(-1, int(action))
	a.EqualError(err, "invalid action: -1")

	action, err = ActionFromString(strconv.Itoa(int(ActionContinue) + 1))
	a.Equal(-1, int(action))
	a.EqualError(err, fmt.Sprintf("invalid action: %d", int(ActionContinue)+1))
}
