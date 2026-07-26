package aceydeucey

import (
	"encoding/json"
	"testing"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// marshalRounds serializes the live rounds slice exactly as EndGame persists it.
// Acey Deucey stores a bare array rather than a wrapper object, and Round has a
// custom MarshalJSON, so this pins both.
func marshalRounds(t *testing.T, rounds []*Round) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(rounds)
	require.NoError(t, err)

	return raw
}

// newTestRound builds a round with a real deck, which Round.MarshalJSON requires
// in order to report the remaining card count.
func newTestRound(playerID int64, pot int, games ...*SingleGame) *Round {
	r := NewRound(DefaultOptions(), playerID, deck.New(), pot)
	r.Games = games
	r.Pot = pot

	return r
}

func TestParseGameLog(t *testing.T) {
	rounds := []*Round{
		newTestRound(1, 200, &SingleGame{
			FirstCard:  deck.CardFromString("5c"),
			MiddleCard: deck.CardFromString("12d"),
			LastCard:   deck.CardFromString("9h"),
			Bet:        Bet{Amount: 100},
			Adjustment: 100,
			Result:     SingleGameResultWon,
		}),
		newTestRound(2, 100, &SingleGame{
			FirstCard:  deck.CardFromString("3c"),
			MiddleCard: deck.CardFromString("4d"),
			Bet:        Bet{Amount: 0},
			Result:     SingleGameResultPass,
		}),
	}

	hand, err := ParseGameLog(marshalRounds(t, rounds))
	require.NoError(t, err)

	assert.Equal(t, 2, hand.Rounds)
	assert.Equal(t, 100, hand.PotCents, "pot reflects the final round")
	assert.Zero(t, hand.AnteCents, "the ante lives in options, which this log omits")
	assert.Len(t, hand.Actions, 2)

	// Player 1 wagered and won.
	p1 := hand.Participant(1)
	assert.True(t, p1.VoluntarilyPlayed)
	assert.False(t, p1.Folded)
	assert.True(t, p1.WentToShowdown)
	assert.True(t, p1.Won)
	assert.Equal(t, 100, p1.AmountWageredCents)
	assert.Equal(t, gamelog.KindBet, hand.Actions[0].Kind)
	assert.Len(t, p1.StartingCards, 2, "the two outer cards are what the bet is made on")

	// Player 2 passed on their only turn, which is the closest thing the game has
	// to giving up a hand.
	p2 := hand.Participant(2)
	assert.False(t, p2.VoluntarilyPlayed)
	assert.True(t, p2.Folded)
	assert.False(t, p2.WentToShowdown)
	assert.Equal(t, gamelog.KindPass, hand.Actions[1].Kind)
	assert.Zero(t, p2.AmountWageredCents)
}

func TestParseGameLog_Loss(t *testing.T) {
	rounds := []*Round{
		newTestRound(1, 300, &SingleGame{
			FirstCard:  deck.CardFromString("5c"),
			MiddleCard: deck.CardFromString("12d"),
			LastCard:   deck.CardFromString("14h"),
			Bet:        Bet{Amount: 150},
			Adjustment: -150,
			Result:     SingleGameResultLost,
		}),
	}

	hand, err := ParseGameLog(marshalRounds(t, rounds))
	require.NoError(t, err)

	p1 := hand.Participant(1)
	assert.True(t, p1.VoluntarilyPlayed)
	assert.True(t, p1.WentToShowdown, "a resolved wager is the showdown")
	assert.False(t, p1.Won)
	assert.Equal(t, 150, p1.AmountWageredCents)
}

func TestParseGameLog_Invalid(t *testing.T) {
	hand, err := ParseGameLog(json.RawMessage(`{"not":"an-array"}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode acey deucey log")
}
