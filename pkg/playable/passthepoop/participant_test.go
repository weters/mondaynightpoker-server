package passthepoop

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParticipant_subtractLife(t *testing.T) {
	p := &Participant{lives: 5}
	assert.Equal(t, 2, p.subtractLife(2))
	assert.Equal(t, 3, p.lives)

	assert.Equal(t, 3, p.subtractLife(0))
	assert.Equal(t, 0, p.lives)

	p.lives = 2

	assert.Equal(t, 2, p.subtractLife(5))
	assert.Equal(t, 0, p.lives)
}
