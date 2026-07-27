package guts

import (
	"encoding/json"
	"testing"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// marshalLog serializes the live gameLog exactly as EndGame persists it, so the
// tests below exercise the real on-disk contract rather than a hand-written
// approximation of it. A field renamed on gameLog breaks these tests, which is the
// point: the parser reads what the game actually writes.
func marshalLog(t *testing.T, log *gameLog) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(log)
	require.NoError(t, err)

	return raw
}

func TestParseGameLog(t *testing.T) {
	log := &gameLog{
		Options:    Options{Ante: 25},
		InitialPot: 50,
		Players:    []int64{1, 2, 3},
		Rounds: []*gameLogRound{
			{
				Round: 1,
				Pot:   75,
				Seats: []*gameLogSeat{
					{PlayerID: 1, Cards: []*deck.Card{deck.CardFromString("14s"), deck.CardFromString("13s")}},
					{PlayerID: 2, Cards: []*deck.Card{deck.CardFromString("2c"), deck.CardFromString("7d")}},
					{PlayerID: 3, Cards: []*deck.Card{deck.CardFromString("9h"), deck.CardFromString("9c")}},
				},
				Decisions: []*gameLogDecision{
					{PlayerID: 1, In: true},
					{PlayerID: 2, In: false},
					{PlayerID: 3, In: true},
				},
				Result: &gameLogShowdown{
					Winners:   []int64{3},
					PlayersIn: []int64{1, 3},
					PotWon:    75,
					FinalHands: []*gameLogSeat{
						{PlayerID: 3, Cards: []*deck.Card{deck.CardFromString("9h"), deck.CardFromString("9c")}},
					},
				},
			},
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	assert.Equal(t, 25, hand.AnteCents)
	assert.Equal(t, 75, hand.PotCents)
	assert.Equal(t, 1, hand.Rounds)
	assert.Len(t, hand.Participants, 3)

	// Player 1 declared in and lost a contested showdown.
	p1 := hand.Participant(1)
	assert.True(t, p1.VoluntarilyPlayed)
	assert.False(t, p1.Folded)
	assert.True(t, p1.WentToShowdown)
	assert.False(t, p1.Won)
	assert.Equal(t, 1, p1.Counts[gamelog.KindStayIn])

	// Player 2 declared out, so they folded the game and never reached a showdown.
	p2 := hand.Participant(2)
	assert.False(t, p2.VoluntarilyPlayed)
	assert.True(t, p2.Folded)
	assert.False(t, p2.WentToShowdown)
	assert.Equal(t, 1, p2.Counts[gamelog.KindDropOut])

	// Player 3 declared in and won.
	p3 := hand.Participant(3)
	assert.True(t, p3.VoluntarilyPlayed)
	assert.True(t, p3.WentToShowdown)
	assert.True(t, p3.Won)
	assert.Len(t, p3.FinalCards, 2)
}

// TestParseGameLog_MultiRound covers the rule that folding is a property of the
// whole game, not of a single round: sitting out one round of a two-round game is
// not a fold.
func TestParseGameLog_MultiRound(t *testing.T) {
	log := &gameLog{
		Options: Options{Ante: 25},
		Players: []int64{1, 2},
		Rounds: []*gameLogRound{
			{
				Round:     1,
				Pot:       50,
				Decisions: []*gameLogDecision{{PlayerID: 1, In: false}, {PlayerID: 2, In: true}},
				Result:    &gameLogShowdown{Winners: []int64{2}, PlayersIn: []int64{2}},
			},
			{
				Round:     2,
				Pot:       100,
				Decisions: []*gameLogDecision{{PlayerID: 1, In: true}, {PlayerID: 2, In: true}},
				Result:    &gameLogShowdown{Winners: []int64{1}, PlayersIn: []int64{1, 2}},
			},
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	assert.Equal(t, 2, hand.Rounds)
	assert.Equal(t, 100, hand.PotCents, "pot should reflect the final round")

	p1 := hand.Participant(1)
	assert.True(t, p1.VoluntarilyPlayed, "declared in on round 2")
	assert.False(t, p1.Folded, "sitting out one round is not folding the game")
	assert.True(t, p1.Won)

	// Round 1 had a single player in, so player 2 won it without a showdown; round 2
	// was contested, which is what sets the flag.
	p2 := hand.Participant(2)
	assert.True(t, p2.WentToShowdown)
}

func TestParseGameLog_Invalid(t *testing.T) {
	hand, err := ParseGameLog(json.RawMessage(`{"rounds":"not-an-array"}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode guts log")
}
