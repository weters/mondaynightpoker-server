package passthepoop

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestDiarrheaEdition_ParticipantWasPassed(t *testing.T) {
	d := &DiarrheaEdition{}
	p := &Participant{
		lives: 2,
		deadCard: false,
	}

	d.ParticipantWasPassed(p, &deck.Card{Rank: deck.King})
	assert.Equal(t, 2, p.lives)
	assert.False(t, p.deadCard)

	d.ParticipantWasPassed(p, &deck.Card{Rank: deck.Ace})
	assert.Equal(t, 1, p.lives)
	assert.True(t, p.deadCard)
}
