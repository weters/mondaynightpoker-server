package aceydeucey

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestNewParticipant(t *testing.T) {
	p := NewParticipant(1, 5)
	assert.Equal(t, int64(1), p.PlayerID)
	assert.Equal(t, -5, p.Balance)
}
