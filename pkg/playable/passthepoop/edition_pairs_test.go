package passthepoop

import (
	"github.com/sirupsen/logrus"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPairsEdition_Name(t *testing.T) {
	p := &PairsEdition{}
	assert.Equal(t, "Pairs", p.Name())
}

func TestPairsEdition_EndRound_NoPairs(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("2c")},
		{PlayerID: 2, lives: 3, card: card("14c")},
		{PlayerID: 3, lives: 3, card: card("4c")},
	}

	std := &PairsEdition{}
	lg, err := std.EndRound(participants)
	assert.NoError(t, err)
	assert.NotNil(t, lg)
	assert.Equal(t, newLoserGroup([]*RoundLoser{
		{
			PlayerID:  2,
			Card:      card("14c"),
			LivesLost: 1,
		},
	}), lg)

	assert.Equal(t, 3, participants[0].lives)
	assert.Equal(t, 2, participants[1].lives)
	assert.Equal(t, 3, participants[2].lives)
}

func TestPairsEdition_EndRound_SinglePair(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("2c")},
		{PlayerID: 2, lives: 3, card: card("2h")},
		{PlayerID: 3, lives: 3, card: card("13c")},
	}

	std := &PairsEdition{}
	lg, err := std.EndRound(participants)
	assert.NoError(t, err)
	assert.NotNil(t, lg)
	assert.Equal(t, newLoserGroup([]*RoundLoser{
		{
			PlayerID:  3,
			Card:      card("13c"),
			LivesLost: 1,
		},
	}), lg)

	assert.Equal(t, 3, participants[0].lives)
	assert.Equal(t, 3, participants[1].lives)
	assert.Equal(t, 2, participants[2].lives)
}

func TestPairsEdition_EndRound_DoublePair(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("2c")},
		{PlayerID: 2, lives: 3, card: card("2h")},
		{PlayerID: 3, lives: 3, card: card("13h")},
		{PlayerID: 4, lives: 3, card: card("13c")},
	}

	std := &PairsEdition{}
	lg, err := std.EndRound(participants)
	assert.NoError(t, err)
	assert.NotNil(t, lg)
	assert.Equal(t, newLoserGroup([]*RoundLoser{
		{
			PlayerID:  1,
			Card:      card("2c"),
			LivesLost: 1,
		},
		{
			PlayerID:  2,
			Card:      card("2h"),
			LivesLost: 1,
		},
	}), lg)

	assert.Equal(t, 2, participants[0].lives)
	assert.Equal(t, 2, participants[1].lives)
	assert.Equal(t, 3, participants[2].lives)
	assert.Equal(t, 3, participants[3].lives)
}

func TestPairsEdition_EndRound_Trips(t *testing.T) {
	participants := []*Participant{
		{PlayerID: 1, lives: 3, card: card("13c")},
		{PlayerID: 2, lives: 3, card: card("3c")},
		{PlayerID: 3, lives: 3, card: card("3d")},
		{PlayerID: 4, lives: 3, card: card("3h")},
		{PlayerID: 5, lives: 3, card: card("2c")},
		{PlayerID: 6, lives: 3, card: card("2h")},
		{PlayerID: 7, lives: 3, card: card("2d")},
	}

	std := &PairsEdition{}
	lg, err := std.EndRound(participants)
	assert.NoError(t, err)
	assert.NotNil(t, lg)
	assert.Equal(t, newLoserGroup([]*RoundLoser{
		{PlayerID: 1, Card: card("13c"), LivesLost: 3},
		{PlayerID: 5, Card: card("2c"), LivesLost: 3},
		{PlayerID: 6, Card: card("2h"), LivesLost: 3},
		{PlayerID: 7, Card: card("2d"), LivesLost: 3},
	}), lg)

	assert.Equal(t, 0, participants[0].lives)
	assert.Equal(t, 3, participants[1].lives)
	assert.Equal(t, 3, participants[2].lives)
	assert.Equal(t, 3, participants[3].lives)
	assert.Equal(t, 0, participants[4].lives)
	assert.Equal(t, 0, participants[5].lives)
	assert.Equal(t, 0, participants[6].lives)
}

func TestPairsEdition_EndRound_Tied(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{Lives: 1, Edition: &PairsEdition{}, Ante: 25})
	assert.NoError(t, err)

	game.idToParticipant[1].card = card("2c")
	game.idToParticipant[2].card = card("2d")

	execOK, _ := createExecFunctions(t, game)
	execOK(1, ActionStay)
	execOK(2, ActionStay)

	assert.NoError(t, game.EndRound())
	assert.Equal(t, 1, game.idToParticipant[1].lives)
	assert.Equal(t, 1, game.idToParticipant[2].lives)

	assert.False(t, game.isGameOver())
	assert.NoError(t, game.nextRound())

	game.idToParticipant[1].card = card("3c")
	game.idToParticipant[2].card = card("2d")

	execOK(2, ActionStay)
	execOK(1, ActionStay)

	assert.NoError(t, game.EndRound())
	assert.Equal(t, 1, game.idToParticipant[1].lives)
	assert.Equal(t, 0, game.idToParticipant[2].lives)
	assert.True(t, game.isGameOver())
}
