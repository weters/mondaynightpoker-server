package passthepoop

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestParticipant_subtractLife(t *testing.T) {
	p := &Participant{lives: 5}

	assert.PanicsWithValue(t, "count cannot be less than 0", func() {
		p.subtractLife(-1)
	})

	assert.Equal(t, 2, p.subtractLife(2))
	assert.Equal(t, 3, p.lives)

	assert.Equal(t, 3, p.subtractLife(0))
	assert.Equal(t, 0, p.lives)

	p.lives = 2

	assert.Equal(t, 2, p.subtractLife(5))
	assert.Equal(t, 0, p.lives)
}

func TestParticipant_MarshalJSON(t *testing.T) {
	p := &Participant{
		PlayerID:  1,
		lives:     2,
		balance:   3,
		card:      deck.CardFromString("4s"),
		deadCard:  true,
		isFlipped: true,
		hasBlock:  true,
	}
	data, err := json.Marshal(p)

	a := assert.New(t)
	a.NoError(err)

	var pJSON participantJSON
	a.NoError(json.Unmarshal(data, &pJSON))

	a.Equal(int64(1), pJSON.PlayerID)
	a.Equal(2, pJSON.Lives)
	a.Equal(3, pJSON.Balance)
	a.Equal("4s", deck.CardToString(pJSON.Card))
	a.True(pJSON.HasBlock)
	a.True(pJSON.IsCardDead)
	a.True(pJSON.IsFlipped)

	p.isFlipped = false
	p.deadCard = false
	p.hasBlock = false
	data, err = json.Marshal(p)
	a.NoError(err)

	pJSON = participantJSON{}
	a.NoError(json.Unmarshal(data, &pJSON))

	a.Nil(pJSON.Card)
	a.False(pJSON.HasBlock)
	a.False(pJSON.IsCardDead)
	a.False(pJSON.IsFlipped)
}
