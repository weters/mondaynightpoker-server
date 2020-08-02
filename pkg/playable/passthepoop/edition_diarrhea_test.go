package passthepoop

import (
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"testing"

	"github.com/stretchr/testify/assert"
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

// nolint:dupl
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
			Order: 0,
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
			Order: 1,
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
			Order: 0,
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

func TestDiarrheaEdition_EndRound_AcePassBack(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, Options{
		Ante:    75,
		Lives:   3,
		Edition: &DiarrheaEdition{},
	})
	assert.NoError(t, err)

	game.participants[0].card = card("4c")  // double'd out
	game.participants[1].card = card("14d") // double'd out
	game.participants[2].card = card("13c") // lose 1
	execOK, _ := createExecFunctions(t, game)

	execOK(1, ActionTrade)  // trade 4 of Clubs
	execOK(2, ActionAccept) // trade back A of Diamonds
	execOK(2, ActionStay)   // stay with 4 of Clubs
	execOK(3, ActionStay)   // stay with K of Clubs

	assert.NoError(t, game.EndRound())
	assert.Equal(t, []*LoserGroup{
		{
			Order: 0,
			RoundLosers: []*RoundLoser{
				{PlayerID: 1, Card: card("14d"), LivesLost: 1},
			},
		},
		{
			Order: 1,
			RoundLosers: []*RoundLoser{
				{PlayerID: 2, Card: card("4c"), LivesLost: 1},
			},
		},
	}, game.loserGroups)

	assert.Equal(t, 2, game.participants[0].lives)
	assert.Equal(t, 2, game.participants[1].lives)
	assert.Equal(t, 3, game.participants[2].lives)
}

func TestDiarrheaEdition_EndRound_AceFromDeck(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4, 5}, Options{
		Ante:    75,
		Lives:   3,
		Edition: &DiarrheaEdition{},
	})
	assert.NoError(t, err)

	game.participants[0].card = card("6c") // double'd out
	game.participants[1].card = card("6d") // double'd out
	game.participants[2].card = card("7c") // lose 1
	game.participants[3].card = card("8c") // safe
	game.participants[4].card = card("3c") // lose 1
	execOK, _ := createExecFunctions(t, game)

	execOK(1, ActionStay)
	execOK(2, ActionStay)
	execOK(3, ActionStay)
	execOK(4, ActionStay)
	execOK(5, ActionGoToDeck)
	game.deck.Cards[0] = card("14c") // ace pass back
	execOK(5, ActionDrawFromDeck)

	assert.NoError(t, game.EndRound())
	assert.Equal(t, []*LoserGroup{
		{
			Order: 0,
			RoundLosers: []*RoundLoser{
				{PlayerID: 5, Card: card("14c"), LivesLost: 1},
			},
		},
		{
			Order: 1,
			RoundLosers: []*RoundLoser{
				{PlayerID: 1, Card: card("6c"), LivesLost: 3},
				{PlayerID: 2, Card: card("6d"), LivesLost: 3},
			},
		},
		{
			Order: 2,
			RoundLosers: []*RoundLoser{
				{PlayerID: 3, Card: card("7c"), LivesLost: 1},
			},
		},
	}, game.loserGroups)

	assert.Equal(t, 0, game.participants[0].lives)
	assert.Equal(t, 0, game.participants[1].lives)
	assert.Equal(t, 2, game.participants[2].lives)
	assert.Equal(t, 3, game.participants[3].lives)
	assert.Equal(t, 2, game.participants[4].lives)
}

func TestDiarrheaEdition_EndRound_TripleAce(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		Ante:    75,
		Lives:   3,
		Edition: &DiarrheaEdition{},
	})
	assert.NoError(t, err)

	game.participants[0].card = card("14c")
	game.participants[1].card = card("14d")
	game.deck.Cards[0] = card("14h")
	execOK, _ := createExecFunctions(t, game)

	execOK(1, ActionTrade)
	execOK(2, ActionAccept)
	execOK(2, ActionGoToDeck)
	execOK(2, ActionDrawFromDeck)

	assert.NoError(t, game.EndRound())
	assert.Equal(t, []*LoserGroup{
		{
			Order: 0,
			RoundLosers: []*RoundLoser{
				{PlayerID: 1, Card: card("14d"), LivesLost: 1},
				{PlayerID: 2, Card: card("14h"), LivesLost: 1},
			},
		},
	}, game.loserGroups)

	assert.Equal(t, 2, game.participants[0].lives)
	assert.Equal(t, 2, game.participants[1].lives)
}

func TestDiarrheaEdition_EndRound_TripleAce_OneLife(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		Ante:    75,
		Lives:   1,
		Edition: &DiarrheaEdition{},
	})
	assert.NoError(t, err)

	game.participants[0].card = card("14c")
	game.participants[1].card = card("14d")
	game.deck.Cards[0] = card("14h")
	execOK, _ := createExecFunctions(t, game)

	execOK(1, ActionTrade)
	execOK(2, ActionAccept)
	execOK(2, ActionGoToDeck)
	execOK(2, ActionDrawFromDeck)

	// game.EndRound eats the ErrMutualDestruction from diarrhea.EndGame
	assert.NoError(t, game.EndRound())
	assert.Equal(t, []*LoserGroup{}, game.loserGroups)

	assert.Equal(t, 1, game.participants[0].lives)
	assert.Equal(t, 1, game.participants[1].lives)
}

func TestDiarrheaEdition_EndRound_DoubleAce_DoubleD(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, Options{
		Ante:    75,
		Lives:   1,
		Edition: &DiarrheaEdition{},
	})
	assert.NoError(t, err)

	game.participants[0].card = card("2c") // ace pass back
	game.participants[1].card = card("14c")
	game.participants[2].card = card("14d")
	game.participants[3].card = card("2d")
	execOK, _ := createExecFunctions(t, game)

	execOK(1, ActionTrade)
	execOK(2, ActionAccept)
	execOK(2, ActionTrade)
	execOK(3, ActionAccept)
	execOK(3, ActionStay)
	execOK(4, ActionStay)

	// game.EndRound eats the ErrMutualDestruction from diarrhea.EndGame
	assert.NoError(t, game.EndRound())
	assert.Equal(t, []*LoserGroup{
		{
			Order: 0,
			RoundLosers: []*RoundLoser{
				{PlayerID: 1, Card: card("14c"), LivesLost: 1},
				{PlayerID: 2, Card: card("14d"), LivesLost: 1},
			},
		}}, game.loserGroups)

	assert.Equal(t, 0, game.participants[0].lives)
	assert.Equal(t, 0, game.participants[1].lives)
	assert.Equal(t, 1, game.participants[2].lives)
	assert.Equal(t, 1, game.participants[3].lives)
}

func TestDiarrheaEdition_EndRound_AceToKing(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, Options{
		Ante:    75,
		Lives:   2,
		Edition: &DiarrheaEdition{},
	})
	assert.NoError(t, err)

	game.participants[0].card = card("3c")
	game.participants[1].card = card("13c")
	game.participants[2].card = card("14c")
	game.participants[2].lives = 1
	game.participants[3].card = card("13d")
	execOK, _ := createExecFunctions(t, game)

	execOK(1, ActionTrade)
	execOK(2, ActionFlipKing)
	execOK(3, ActionTrade)
	execOK(4, ActionFlipKing)

	assert.NoError(t, game.EndRound())
	assert.Equal(t, []*LoserGroup{
		{
			Order: 0,
			RoundLosers: []*RoundLoser{
				{PlayerID: 3, Card: card("14c"), LivesLost: 1},
			},
		},
	}, game.loserGroups)

	assert.Equal(t, 2, game.participants[0].lives)
	assert.Equal(t, 2, game.participants[1].lives)
	assert.Equal(t, 0, game.participants[2].lives)
	assert.Equal(t, 2, game.participants[3].lives)

	assert.NoError(t, game.nextRound())
	assert.Equal(t, []int64{2, 4, 1}, getPlayerIDsFromGame(game))
}

func TestDiarrheaEdition_EndRound_AcePassBack_2(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4, 5}, Options{
		Ante:    75,
		Lives:   2,
		Edition: &DiarrheaEdition{},
	})
	assert.NoError(t, err)

	execOK, _ := createExecFunctions(t, game)

	game.participants[0].card = card("8c")
	game.participants[1].card = card("4c")
	game.participants[2].card = card("14c")
	game.participants[3].card = card("2c")
	game.participants[4].card = card("10c")

	execOK(1, ActionStay)
	execOK(2, ActionTrade)
	execOK(3, ActionAccept)
	execOK(3, ActionTrade)
	execOK(4, ActionAccept)
	execOK(4, ActionStay)
	execOK(5, ActionStay)

	assert.NoError(t, game.EndRound())
	assert.Equal(t, []*LoserGroup{
		{
			Order: 0,
			RoundLosers: []*RoundLoser{
				{PlayerID: 2, Card: card("14c"), LivesLost: 1},
			},
		},
		{
			Order: 1,
			RoundLosers: []*RoundLoser{
				{PlayerID: 3, Card: card("2c"), LivesLost: 1},
			},
		},
	}, game.loserGroups)

	assert.Equal(t, 2, game.participants[0].lives)
	assert.Equal(t, 1, game.participants[1].lives)
	assert.Equal(t, 1, game.participants[2].lives)
	assert.Equal(t, 2, game.participants[3].lives)
	assert.Equal(t, 2, game.participants[4].lives)

	assert.NoError(t, game.nextRound())
	assert.Equal(t, []int64{2, 3, 4, 5, 1}, getPlayerIDsFromGame(game))

	game.participants[0].card = card("14c") // 2
	game.participants[1].card = card("13c") // 3
	game.participants[2].card = card("2c")  // 4
	game.participants[3].card = card("3c")  // 5
	game.participants[4].card = card("4c")  // 6

	execOK(2, ActionTrade)
	execOK(3, ActionFlipKing)
	execOK(4, ActionStay)
	execOK(5, ActionStay)
	execOK(1, ActionStay)

	assert.NoError(t, game.EndRound())
	assert.Equal(t, []*LoserGroup{
		{
			Order: 0,
			RoundLosers: []*RoundLoser{
				{PlayerID: 2, Card: card("14c"), LivesLost: 1},
			},
		},
	}, game.loserGroups)

	assert.False(t, game.participants[0].deadCard)
	assert.Equal(t, 0, game.participants[0].lives)
	assert.Equal(t, 1, game.participants[1].lives)
	assert.Equal(t, 2, game.participants[2].lives)
	assert.Equal(t, 2, game.participants[3].lives)
	assert.Equal(t, 2, game.participants[4].lives)
}

func TestDiarrheaEdition_EndRound_AceFromDeck_DoubleD_Safe(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, Options{
		Ante:    75,
		Lives:   3,
		Edition: &DiarrheaEdition{},
	})
	assert.NoError(t, err)

	game.participants[0].card = card("6c") // double'd out
	game.participants[1].card = card("6d") // double'd out
	game.participants[2].card = card("7c") // safe
	game.participants[3].card = card("3c") // lose 1
	execOK, _ := createExecFunctions(t, game)

	execOK(1, ActionStay)
	execOK(2, ActionStay)
	execOK(3, ActionStay)
	execOK(4, ActionGoToDeck)
	game.deck.Cards[0] = card("14c") // ace pass back
	execOK(4, ActionDrawFromDeck)

	assert.NoError(t, game.EndRound())
	assert.Equal(t, []*LoserGroup{
		{
			Order: 0,
			RoundLosers: []*RoundLoser{
				{PlayerID: 4, Card: card("14c"), LivesLost: 1},
			},
		},
		{
			Order: 1,
			RoundLosers: []*RoundLoser{
				{PlayerID: 1, Card: card("6c"), LivesLost: 3},
				{PlayerID: 2, Card: card("6d"), LivesLost: 3},
			},
		},
	}, game.loserGroups)

	assert.Equal(t, 0, game.participants[0].lives)
	assert.Equal(t, 0, game.participants[1].lives)
	assert.Equal(t, 3, game.participants[2].lives)
	assert.Equal(t, 2, game.participants[3].lives)
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
	assert.Equal(t, 3, participants[0].lives)
	assert.Equal(t, 3, participants[1].lives)
}
