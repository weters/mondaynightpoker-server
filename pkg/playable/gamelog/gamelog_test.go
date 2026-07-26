package gamelog

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawID_UnmarshalJSON(t *testing.T) {
	// The object form is what the game packages' MarshalJSON implementations
	// actually write; the bare string is the form a plain string field produces.
	testCases := []struct {
		name string
		raw  string
		want RawID
	}{
		{"object form", `{"id":"fold","name":"Fold"}`, "fold"},
		{"object form with extra fields", `{"id":"pineapple","name":"Pineapple","holeCards":3}`, "pineapple"},
		{"bare string", `"raise"`, "raise"},
		{"empty string", `""`, ""},
		{"object without id", `{"name":"Fold"}`, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got RawID
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &got))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRawID_UnmarshalJSON_Invalid(t *testing.T) {
	var got RawID
	err := json.Unmarshal([]byte(`[1,2,3]`), &got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode identifier")
}

func TestRawID_Kind(t *testing.T) {
	for raw, want := range map[RawID]Kind{
		"fold":    KindFold,
		"check":   KindCheck,
		"call":    KindCall,
		"bet":     KindBet,
		"raise":   KindRaise,
		"discard": KindDiscard,
		"trade":   KindTrade,
	} {
		got, ok := raw.Kind()
		assert.True(t, ok, "%s should be known", raw)
		assert.Equal(t, want, got)
	}

	// An identifier from a newer game version must not be mistaken for a known one.
	got, ok := RawID("teleport").Kind()
	assert.False(t, ok)
	assert.Equal(t, Kind(""), got)
}

func TestKind_IsAggressive(t *testing.T) {
	assert.True(t, KindBet.IsAggressive())
	assert.True(t, KindRaise.IsAggressive())
	assert.False(t, KindCall.IsAggressive())
	assert.False(t, KindCheck.IsAggressive())
	assert.False(t, KindFold.IsAggressive())
}

func TestHand_Participant(t *testing.T) {
	hand := &Hand{}

	first := hand.Participant(7)
	assert.Equal(t, int64(7), first.PlayerID)
	assert.Len(t, hand.Participants, 1)

	// Asking again returns the same record rather than adding a duplicate.
	again := hand.Participant(7)
	assert.Same(t, first, again)
	assert.Len(t, hand.Participants, 1)

	hand.Participant(8)
	assert.Len(t, hand.Participants, 2)
}

func TestHand_AddAction(t *testing.T) {
	hand := &Hand{}

	hand.AddAction(&Action{PlayerID: 1, Kind: KindBet, AmountCents: 50})
	hand.AddAction(&Action{PlayerID: 1, Kind: KindCall, AmountCents: 25})
	hand.AddAction(&Action{PlayerID: 2, Kind: KindFold})

	// Sequence is assigned in insertion order.
	require.Len(t, hand.Actions, 3)
	assert.Equal(t, 0, hand.Actions[0].Sequence)
	assert.Equal(t, 2, hand.Actions[2].Sequence)

	p1 := hand.Participant(1)
	assert.Equal(t, 75, p1.AmountWageredCents)
	assert.Equal(t, 1, p1.Counts[KindBet])
	assert.Equal(t, 1, p1.Counts[KindCall])
	assert.False(t, p1.Folded)

	// AddAction records the fold but leaves VoluntarilyPlayed to the parser, which
	// is the only thing that knows which contributions were forced.
	p2 := hand.Participant(2)
	assert.True(t, p2.Folded)
	assert.False(t, p2.VoluntarilyPlayed)
}

func TestHand_AddAction_DropOutFolds(t *testing.T) {
	hand := &Hand{}
	hand.AddAction(&Action{PlayerID: 1, Kind: KindDropOut})

	assert.True(t, hand.Participant(1).Folded, "dropping out is folding")
}

func TestHand_ApplyBettingActions(t *testing.T) {
	hand := &Hand{}
	hand.ApplyBettingActions([]BettingAction{
		{Street: "preflop", PlayerID: 1, Action: "raise", Amount: 100},
		{Street: "preflop", PlayerID: 2, Action: "call", Amount: 100},
		{Street: "preflop", PlayerID: 3, Action: "check"},
		{Street: "preflop", PlayerID: 4, Action: "fold"},
		// An action from a newer game version is skipped, not fatal.
		{Street: "preflop", PlayerID: 5, Action: "teleport"},
	})

	require.Len(t, hand.Actions, 4, "the unknown action is skipped")

	assert.True(t, hand.Participant(1).VoluntarilyPlayed, "raising is voluntary")
	assert.True(t, hand.Participant(2).VoluntarilyPlayed, "calling is voluntary")
	assert.False(t, hand.Participant(3).VoluntarilyPlayed, "checking is free")
	assert.False(t, hand.Participant(4).VoluntarilyPlayed, "folding is not playing")
	assert.True(t, hand.Participant(4).Folded)
}

func TestHand_ApplyBettingActionsFunc(t *testing.T) {
	hand := &Hand{}
	hand.ApplyBettingActionsFunc([]BettingAction{
		{Street: "third-street", PlayerID: 1, Action: "bet", Amount: 50},
		{Street: "third-street", PlayerID: 2, Action: "flip-mushroom"},
		{Street: "third-street", PlayerID: 2, Action: "call", Amount: 50},
		// Still unknown to both the shared vocabulary and the fallback.
		{Street: "third-street", PlayerID: 3, Action: "teleport"},
	}, func(raw RawID) (Kind, bool) {
		if raw == "flip-mushroom" {
			return KindPlayCard, true
		}

		return "", false
	})

	// The fallback action keeps its position between the two betting actions.
	require.Len(t, hand.Actions, 3)
	assert.Equal(t, KindPlayCard, hand.Actions[1].Kind)
	assert.Equal(t, int64(2), hand.Actions[1].PlayerID)
	assert.Equal(t, "third-street", hand.Actions[1].Street)
	assert.Equal(t, 1, hand.Actions[1].Sequence)
	assert.Equal(t, KindCall, hand.Actions[2].Kind)

	// The fallback is never what makes a player voluntarily played; the call is.
	assert.True(t, hand.Participant(2).VoluntarilyPlayed)
	assert.Equal(t, 50, hand.Participant(2).AmountWageredCents)
	assert.Nil(t, hand.FindParticipant(3), "the unresolved action added nobody")
}

func TestHand_ApplyBettingActionsFunc_FallbackIsNotVoluntary(t *testing.T) {
	hand := &Hand{}
	// A fallback that resolves to a kind the betting path would treat as voluntary
	// must still not set the flag: only the shared vocabulary can.
	hand.ApplyBettingActionsFunc([]BettingAction{
		{PlayerID: 1, Action: "flip-mushroom", Amount: 0},
	}, func(RawID) (Kind, bool) {
		return KindRaise, true
	})

	require.Len(t, hand.Actions, 1)
	assert.False(t, hand.Participant(1).VoluntarilyPlayed)
}

func TestHand_ApplyBettingActionsFunc_NilFallback(t *testing.T) {
	hand := &Hand{}
	hand.ApplyBettingActionsFunc([]BettingAction{
		{PlayerID: 1, Action: "check"},
		{PlayerID: 2, Action: "flip-mushroom"},
	}, nil)

	require.Len(t, hand.Actions, 1, "without a fallback the unknown action is skipped")
	assert.Equal(t, KindCheck, hand.Actions[0].Kind)
}

func TestHand_ResolveShowdown(t *testing.T) {
	t.Run("contested", func(t *testing.T) {
		hand := &Hand{}
		hand.Participant(1)
		hand.Participant(2)
		hand.Participant(3)

		hand.ResolveShowdown(
			map[int64]bool{3: true},
			map[int64]int{1: 300},
		)

		assert.True(t, hand.Participant(1).WentToShowdown)
		assert.True(t, hand.Participant(1).Won)
		assert.True(t, hand.Participant(2).WentToShowdown)
		assert.False(t, hand.Participant(2).Won)
		assert.True(t, hand.Participant(3).Folded)
		assert.False(t, hand.Participant(3).WentToShowdown)
	})

	t.Run("uncontested", func(t *testing.T) {
		hand := &Hand{}
		hand.Participant(1)
		hand.Participant(2)

		hand.ResolveShowdown(
			map[int64]bool{2: true},
			map[int64]int{1: 100},
		)

		// Everyone else folded, so the pot was won without a showdown.
		assert.True(t, hand.Participant(1).Won)
		assert.False(t, hand.Participant(1).WentToShowdown)
	})
}
