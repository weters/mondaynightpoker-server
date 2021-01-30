package aceydeucey

import (
	"github.com/stretchr/testify/assert"
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
