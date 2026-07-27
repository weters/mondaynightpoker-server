package sevencard

import (
	"encoding/json"
	"testing"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// marshalLog serializes the live gameLog exactly as EndGame persists it, so these
// tests exercise the real on-disk contract rather than a hand-written
// approximation of it.
func marshalLog(t *testing.T, log *gameLog) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(log)
	require.NoError(t, err)

	return raw
}

func TestParseGameLog(t *testing.T) {
	log := &gameLog{
		Variant: "baseball",
		Ante:    25,
		Seats: []*gameLogSeat{
			{PlayerID: 1, TableStake: 1000, SeatIndex: 0},
			{PlayerID: 2, TableStake: 1000, SeatIndex: 1},
		},
		Deals: []*gameLogDeal{
			{
				Street: "initial",
				Cards: []*gameLogDealCard{
					{PlayerID: 1, Card: deck.CardFromString("14s"), FaceUp: false},
					{PlayerID: 1, Card: deck.CardFromString("13s"), FaceUp: true},
					{PlayerID: 2, Card: deck.CardFromString("3c"), FaceUp: false},
				},
			},
			{
				Street: "fourth-street",
				Cards: []*gameLogDealCard{
					{PlayerID: 1, Card: deck.CardFromString("12s"), FaceUp: true},
				},
			},
		},
		Actions: []*gameLogAction{
			{Street: "third-street", PlayerID: 1, Action: ActionBet, Amount: 50},
			{Street: "third-street", PlayerID: 2, Action: ActionCall, Amount: 50},
			{Street: "fourth-street", PlayerID: 1, Action: ActionBet, Amount: 100},
			{Street: "fourth-street", PlayerID: 2, Action: ActionFold},
		},
		FinalState: GameState{
			Pot:     225,
			Winners: map[int64]int{1: 225},
			Participants: []*participantJSON{
				{PlayerID: 1, Hand: deck.Hand{deck.CardFromString("14s"), deck.CardFromString("13s")}},
				{PlayerID: 2, DidFold: true},
			},
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	assert.Equal(t, "baseball", hand.Variant)
	assert.Equal(t, 25, hand.AnteCents)
	assert.Equal(t, 225, hand.PotCents)
	assert.Len(t, hand.Actions, 4)

	// Seven Card deals a card per street; starting cards come from the initial deal
	// only, so player 1's fourth-street card must not appear there.
	p1 := hand.Participant(1)
	assert.Len(t, p1.StartingCards, 2)
	assert.True(t, p1.VoluntarilyPlayed)
	assert.True(t, p1.Won)
	assert.False(t, p1.WentToShowdown, "the opponent folded, so nothing was shown down")
	assert.Equal(t, 150, p1.AmountWageredCents)

	p2 := hand.Participant(2)
	assert.Len(t, p2.StartingCards, 1)
	assert.True(t, p2.VoluntarilyPlayed, "called before folding later")
	assert.True(t, p2.Folded)
	assert.Equal(t, gamelog.KindFold, hand.Actions[3].Kind)
}

func TestParseGameLog_Showdown(t *testing.T) {
	log := &gameLog{
		Variant: "stud",
		Ante:    25,
		Seats:   []*gameLogSeat{{PlayerID: 1}, {PlayerID: 2}},
		Actions: []*gameLogAction{
			{Street: "river", PlayerID: 1, Action: ActionCheck},
			{Street: "river", PlayerID: 2, Action: ActionCheck},
		},
		FinalState: GameState{
			Pot:     100,
			Winners: map[int64]int{2: 100},
			Participants: []*participantJSON{
				{PlayerID: 1},
				{PlayerID: 2},
			},
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	assert.True(t, hand.Participant(1).WentToShowdown)
	assert.False(t, hand.Participant(1).Won)
	assert.True(t, hand.Participant(2).WentToShowdown)
	assert.True(t, hand.Participant(2).Won)
}

// TestParseGameLog_VariantActions covers the Chiggs mushroom/antidote moves, which
// Game.Action records into the same stream as the betting. They must survive
// parsing with their position intact, since where they fall relative to the betting
// is the only thing that says when the mushroom was flipped.
func TestParseGameLog_VariantActions(t *testing.T) {
	log := &gameLog{
		Variant: "chiggs",
		Ante:    25,
		Seats:   []*gameLogSeat{{PlayerID: 1}, {PlayerID: 2}},
		Actions: []*gameLogAction{
			{Street: "fourth-street", PlayerID: 1, Action: ActionBet, Amount: 50},
			{Street: "fourth-street", PlayerID: 2, Action: ActionFlipMushroom},
			{Street: "fourth-street", PlayerID: 1, Action: ActionPlayAntidote},
			{Street: "fourth-street", PlayerID: 2, Action: ActionCall, Amount: 50},
		},
		FinalState: GameState{
			Pot:     150,
			Winners: map[int64]int{1: 150},
			Participants: []*participantJSON{
				{PlayerID: 1},
				{PlayerID: 2},
			},
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	require.Len(t, hand.Actions, 4, "no variant action is dropped")

	// The two variant actions keep the sequence positions they were recorded in,
	// interleaved with the betting rather than appended after it.
	assert.Equal(t, gamelog.KindBet, hand.Actions[0].Kind)
	assert.Equal(t, gamelog.KindPlayCard, hand.Actions[1].Kind)
	assert.Equal(t, gamelog.KindPlayCard, hand.Actions[2].Kind)
	assert.Equal(t, gamelog.KindCall, hand.Actions[3].Kind)

	for i, a := range hand.Actions {
		assert.Equal(t, i, a.Sequence)
		assert.Equal(t, "fourth-street", a.Street)
	}

	assert.Equal(t, int64(2), hand.Actions[1].PlayerID, "player 2 flipped the mushroom")
	assert.Equal(t, int64(1), hand.Actions[2].PlayerID, "player 1 played the antidote")

	// Neither move puts money in the pot, so neither shows up in the wagered totals
	// and neither can be what makes a player voluntarily played.
	assert.Zero(t, hand.Actions[1].AmountCents)
	assert.Zero(t, hand.Actions[2].AmountCents)

	p1 := hand.Participant(1)
	assert.Equal(t, 50, p1.AmountWageredCents)
	assert.True(t, p1.VoluntarilyPlayed, "the bet is what qualifies, not the antidote")
	assert.Equal(t, 1, p1.Counts[gamelog.KindPlayCard])
	assert.False(t, p1.Folded)

	p2 := hand.Participant(2)
	assert.Equal(t, 50, p2.AmountWageredCents)
	assert.True(t, p2.VoluntarilyPlayed, "the call is what qualifies, not the mushroom")
	assert.Equal(t, 1, p2.Counts[gamelog.KindPlayCard])
}

// TestParseGameLog_VariantActionOnly proves the variant moves alone never make a
// player count as having voluntarily played: only money in by choice does that.
func TestParseGameLog_VariantActionOnly(t *testing.T) {
	log := &gameLog{
		Variant: "chiggs",
		Ante:    25,
		Seats:   []*gameLogSeat{{PlayerID: 1}, {PlayerID: 2}},
		Actions: []*gameLogAction{
			{Street: "third-street", PlayerID: 1, Action: ActionFlipMushroom},
			{Street: "third-street", PlayerID: 2, Action: ActionPlayAntidote},
			{Street: "third-street", PlayerID: 1, Action: ActionCheck},
			{Street: "third-street", PlayerID: 2, Action: ActionCheck},
		},
		FinalState: GameState{
			Pot:     50,
			Winners: map[int64]int{1: 50},
			Participants: []*participantJSON{
				{PlayerID: 1},
				{PlayerID: 2},
			},
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	require.Len(t, hand.Actions, 4)

	for _, id := range []int64{1, 2} {
		p := hand.Participant(id)
		assert.False(t, p.VoluntarilyPlayed, "player %d only checked", id)
		assert.Zero(t, p.AmountWageredCents)
		assert.False(t, p.Folded)
		assert.True(t, p.WentToShowdown)
	}
}

// TestVariantActionKind pins the mapping itself, including the unknown-identifier
// case that keeps a log from a newer game version readable.
func TestVariantActionKind(t *testing.T) {
	for _, a := range []Action{ActionFlipMushroom, ActionPlayAntidote} {
		kind, ok := variantActionKind(gamelog.RawID(a))
		assert.True(t, ok, "%s should map", a)
		assert.Equal(t, gamelog.KindPlayCard, kind)
	}

	kind, ok := variantActionKind(gamelog.RawID("summon-chigg"))
	assert.False(t, ok)
	assert.Equal(t, gamelog.Kind(""), kind)
}

func TestParseGameLog_Invalid(t *testing.T) {
	hand, err := ParseGameLog(json.RawMessage(`{"seats":"nope"}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode seven card log")
}
