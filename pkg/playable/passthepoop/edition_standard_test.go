package passthepoop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// nolint:dupl
func TestStandardEdition_EndRound_SingleLoser(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("2c")},
		{PlayerID: 2, lives: 3, card: card("14c")},
		{PlayerID: 3, lives: 3, card: card("4c")},
	}

	std := &StandardEdition{}
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

func TestStandardEdition_EndRound_MultiLoser(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	execOk, _ := createExecFunctions(t, game)
	dealCards(game, "3c", "4c", "3c")
	execOk(1, ActionStay)
	execOk(2, ActionStay)
	execOk(3, ActionStay)
	assert.NoError(t, game.EndRound())
	assert.EqualError(t, game.EndRound(), "you cannot end the round multiple times")
	livesEqual(t, game, map[int64]int{
		1: 2,
		2: 3,
		3: 2,
	})
}

func TestStandardEdition_EndRound_AllLost_WithLife(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 2, card: card("3c")},
		{PlayerID: 2, lives: 1, card: card("3h")},
	}

	std := &StandardEdition{}
	lg, err := std.EndRound(participants)
	assert.NoError(t, err)
	assert.NotNil(t, lg)
	assert.Equal(t, 1, len(lg))
	assert.Equal(t, 2, len(lg[0].RoundLosers))
	assert.Equal(t, &RoundLoser{
		PlayerID:  1,
		Card:      card("3c"),
		LivesLost: 1,
	}, lg[0].RoundLosers[0])
	assert.Equal(t, &RoundLoser{
		PlayerID:  2,
		Card:      card("3h"),
		LivesLost: 1,
	}, lg[0].RoundLosers[1])

	assert.Equal(t, 1, participants[0].lives)
	assert.Equal(t, 0, participants[1].lives)
}

func TestStandardEdition_EndRound_AllLost_Stalemate(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 1, card: card("3c")},
		{PlayerID: 2, lives: 1, card: card("3h")},
	}

	std := &StandardEdition{}
	lg, err := std.EndRound(participants)
	assert.Equal(t, ErrMutualDestruction, err)
	assert.Nil(t, lg)

	assert.Equal(t, 1, participants[0].lives)
	assert.Equal(t, 1, participants[1].lives)
}
