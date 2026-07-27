package passthepoop

import (
	"encoding/json"
	"testing"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// marshalLog serializes the live GameLog exactly as EndGame persists it, so these
// tests exercise the real on-disk contract rather than a hand-written
// approximation of it.
func marshalLog(t *testing.T, log *GameLog) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(log)
	require.NoError(t, err)

	return raw
}

func TestParseGameLog(t *testing.T) {
	log := &GameLog{
		Edition: "diarrhea",
		Pot:     300,
		Ante:    25,
		Lives:   3,
		Players: []int64{1, 2, 3},
		Winner:  2,
		Rounds: []*GameLogRound{
			{
				Round: 0,
				StartingHand: []*GameLogHand{
					{PlayerID: 1, Card: deck.CardFromString("2c")},
					{PlayerID: 2, Card: deck.CardFromString("13s")},
					{PlayerID: 3, Card: deck.CardFromString("8h")},
				},
				GameActions: []*GameActionDetails{
					{GameAction: ActionTrade, PlayerID: 1, SecondaryPlayerID: 2},
					{GameAction: ActionAccept, PlayerID: 2},
					{GameAction: ActionStay, PlayerID: 3},
				},
				LoserGroups: []*LoserGroup{
					{Order: 0, RoundLosers: []*RoundLoser{{PlayerID: 1}}},
				},
			},
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	assert.Equal(t, "diarrhea", hand.Variant)
	assert.Equal(t, 25, hand.AnteCents)
	assert.Equal(t, 300, hand.PotCents)
	assert.Equal(t, 1, hand.Rounds)
	assert.Len(t, hand.Actions, 3)

	// GameAction is an int enum written as an {"id","name"} object with no matching
	// UnmarshalJSON, so this pins that the object form decodes by id.
	assert.Equal(t, gamelog.KindTrade, hand.Actions[0].Kind)
	assert.Equal(t, gamelog.KindStayIn, hand.Actions[2].Kind)

	// Trading is a choice; accepting a trade forced on you is not.
	assert.True(t, hand.Participant(1).VoluntarilyPlayed)
	assert.False(t, hand.Participant(2).VoluntarilyPlayed)
	assert.True(t, hand.Participant(3).VoluntarilyPlayed)

	// Every player dealt into a round contests it: there is nothing to fold.
	for _, id := range []int64{1, 2, 3} {
		assert.True(t, hand.Participant(id).WentToShowdown, "player %d", id)
		assert.False(t, hand.Participant(id).Folded, "player %d", id)
	}

	assert.True(t, hand.Participant(2).Won)
	assert.False(t, hand.Participant(1).Won)
}

func TestParseGameLog_Invalid(t *testing.T) {
	hand, err := ParseGameLog(json.RawMessage(`{"rounds":7}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode pass the poop log")
}
