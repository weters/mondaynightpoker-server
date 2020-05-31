package passthepoop

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestDiarrheaEdition_Name(t *testing.T) {
	d := &DiarrheaEdition{}
	assert.Equal(t, "Diarrhea", d.Name())
}

func TestDiarrheaEdition_ParticipantWasPassed(t *testing.T) {
	d := &DiarrheaEdition{}
	p := &Participant{
		lives:    2,
		deadCard: false,
	}

	d.ParticipantWasPassed(p, &deck.Card{Rank: deck.King})
	assert.Equal(t, 2, p.lives)
	assert.False(t, p.deadCard)

	d.ParticipantWasPassed(p, &deck.Card{Rank: deck.Ace})
	assert.Equal(t, 1, p.lives)
	assert.True(t, p.deadCard)
}

func TestDiarrheaEdition_EndRound_NormalFinish(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("2c")},
		{PlayerID: 2, lives: 3, card: card("14c")},
		{PlayerID: 3, lives: 3, card: card("4c")},
	}

	std := &DiarrheaEdition{}
	lg, err := std.EndRound(participants)
	assert.NoError(t, err)
	assert.NotNil(t, lg)
	assert.Equal(t, 1, len(lg))
	assert.Equal(t, 1, len(lg[0].RoundLosers))
	assert.Equal(t, &RoundLoser{
		PlayerID:  2,
		Card:      card("14c"),
		LivesLost: 1,
	}, lg[0].RoundLosers[0])

	assert.Equal(t, 3, participants[0].lives)
	assert.Equal(t, 2, participants[1].lives)
	assert.Equal(t, 3, participants[2].lives)
}

func TestDiarrheaEdition_EndRound_SingleDouble(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("2c")},
		{PlayerID: 2, lives: 3, card: card("4c")},
		{PlayerID: 3, lives: 3, card: card("2c")},
		{PlayerID: 4, lives: 3, card: card("3c")},
	}

	std := &DiarrheaEdition{}
	lg, err := std.EndRound(participants)
	assert.NoError(t, err)
	assert.NotNil(t, lg)
	assert.Equal(t, 2, len(lg))
	assert.Equal(t, 2, len(lg[0].RoundLosers))
	assert.Equal(t, 1, len(lg[1].RoundLosers))
	assert.Equal(t, &RoundLoser{
		PlayerID:  1,
		Card:      card("2c"),
		LivesLost: 3,
	}, lg[0].RoundLosers[0])
	assert.Equal(t, &RoundLoser{
		PlayerID:  3,
		Card:      card("2c"),
		LivesLost: 3,
	}, lg[0].RoundLosers[1])
	assert.Equal(t, &RoundLoser{
		PlayerID:  4,
		Card:      card("3c"),
		LivesLost: 1,
	}, lg[1].RoundLosers[0])

	assert.Equal(t, 0, participants[0].lives)
	assert.Equal(t, 3, participants[1].lives)
	assert.Equal(t, 0, participants[2].lives)
	assert.Equal(t, 2, participants[3].lives)
}

func TestDiarrheaEdition_EndRound_DoubleDouble(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("2c")},
		{PlayerID: 2, lives: 3, card: card("3c")},
		{PlayerID: 3, lives: 3, card: card("4c")},
		{PlayerID: 4, lives: 3, card: card("3c")},
		{PlayerID: 5, lives: 3, card: card("2c")},
	}

	std := &DiarrheaEdition{}
	lg, err := std.EndRound(participants)
	assert.NoError(t, err)
	assert.NotNil(t, lg)
	assert.Equal(t, []*LoserGroup{
		{
			Order:       0,
			RoundLosers: []*RoundLoser{
				{
					PlayerID:  1,
					Card:      card("2c"),
					LivesLost: 3,
				},
				{
					PlayerID:  5,
					Card:      card("2c"),
					LivesLost: 3,
				},
			},
		},
		{
			Order:       1,
			RoundLosers: []*RoundLoser{
				{
					PlayerID:  2,
					Card:      card("3c"),
					LivesLost: 3,
				},
				{
					PlayerID:  4,
					Card:      card("3c"),
					LivesLost: 3,
				},
			},
		},
	}, lg)

	assert.Equal(t, 0, participants[0].lives)
	assert.Equal(t, 0, participants[1].lives)
	assert.Equal(t, 3, participants[2].lives)
	assert.Equal(t, 0, participants[3].lives)
	assert.Equal(t, 0, participants[4].lives)
}

func TestDiarrheaEdition_EndRound_DoubleDoubleBail(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("2c")},
		{PlayerID: 2, lives: 3, card: card("3c")},
		{PlayerID: 3, lives: 3, card: card("3c")},
		{PlayerID: 4, lives: 3, card: card("2c")},
	}

	std := &DiarrheaEdition{}
	lg, err := std.EndRound(participants)
	assert.NoError(t, err)
	assert.NotNil(t, lg)
	assert.Equal(t, []*LoserGroup{
		{
			Order:       0,
			RoundLosers: []*RoundLoser{
				{
					PlayerID:  1,
					Card:      card("2c"),
					LivesLost: 3,
				},
				{
					PlayerID:  4,
					Card:      card("2c"),
					LivesLost: 3,
				},
			},
		},
	}, lg)

	assert.Equal(t, 0, participants[0].lives)
	assert.Equal(t, 3, participants[1].lives)
	assert.Equal(t, 3, participants[2].lives)
	assert.Equal(t, 0, participants[3].lives)
}

func TestDiarrheaEdition_EndRound_MutualDestruction(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("2c")},
		{PlayerID: 2, lives: 3, card: card("2c")},
	}

	std := &DiarrheaEdition{}
	lg, err := std.EndRound(participants)
	assert.Nil(t, lg)
	assert.Equal(t, ErrMutualDestruction, err)
}
