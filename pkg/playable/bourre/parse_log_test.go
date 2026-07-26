package bourre

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
// approximation of it. Result in particular has no JSON tags, so this pins the
// Go-field-name encoding the parser relies on.
func marshalLog(t *testing.T, log *gameLog) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(log)
	require.NoError(t, err)

	return raw
}

func TestParseGameLog(t *testing.T) {
	winner := NewPlayer(1)
	folder := NewPlayer(3)

	log := &gameLog{
		Options:    Options{Ante: 50},
		Ante:       50,
		InitialPot: 150,
		TrumpCard:  deck.CardFromString("14s"),
		Seats: []*gameLogSeat{
			{PlayerID: 1, StartingHand: []*deck.Card{deck.CardFromString("13s"), deck.CardFromString("12s")}},
			{PlayerID: 2, StartingHand: []*deck.Card{deck.CardFromString("3c"), deck.CardFromString("8h")}},
			{PlayerID: 3, StartingHand: []*deck.Card{deck.CardFromString("2d"), deck.CardFromString("5c")}},
		},
		Discards: []*gameLogDiscard{
			{PlayerID: 1, Discarded: []*deck.Card{deck.CardFromString("12s")}, NewCards: []*deck.Card{deck.CardFromString("11s")}},
			{PlayerID: 2, Discarded: []*deck.Card{}},
			{PlayerID: 3, Folded: true},
		},
		Tricks: []*gameLogTrick{
			{
				Number: 1,
				Plays: []*gameLogPlay{
					{PlayerID: 1, Card: deck.CardFromString("13s")},
					{PlayerID: 2, Card: deck.CardFromString("3c")},
				},
				WinnerID: 1,
			},
		},
		Result: &Result{
			Winners:       []*Player{winner},
			Folded:        []*Player{folder},
			WinningAmount: 150,
			OldPot:        150,
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	assert.Equal(t, 50, hand.AnteCents)
	assert.Equal(t, 150, hand.PotCents)
	assert.Equal(t, 1, hand.Rounds)
	require.Len(t, hand.Board, 1, "the trump card is the shared card")
	assert.Equal(t, 14, hand.Board[0].Rank)

	p1 := hand.Participant(1)
	assert.True(t, p1.VoluntarilyPlayed)
	assert.False(t, p1.Folded)
	assert.True(t, p1.WentToShowdown)
	assert.True(t, p1.Won)
	assert.Equal(t, 1, p1.Counts[gamelog.KindDiscard])
	assert.Equal(t, 1, p1.Counts[gamelog.KindPlayCard])

	p2 := hand.Participant(2)
	assert.True(t, p2.VoluntarilyPlayed)
	assert.True(t, p2.WentToShowdown)
	assert.False(t, p2.Won)

	// Folding in Bourré happens at the trade-in phase, before any trick.
	p3 := hand.Participant(3)
	assert.True(t, p3.Folded)
	assert.False(t, p3.VoluntarilyPlayed)
	assert.False(t, p3.WentToShowdown)
	assert.Equal(t, 1, p3.Counts[gamelog.KindDropOut])
}

// TestParseGameLog_Continuation covers a chained hand: the parent hands count
// toward Rounds without their participation being merged into the final hand.
func TestParseGameLog_Continuation(t *testing.T) {
	log := &gameLog{
		Parent: &gameLog{
			Parent: &gameLog{Ante: 50},
			Ante:   50,
		},
		Ante:  50,
		Seats: []*gameLogSeat{{PlayerID: 1}, {PlayerID: 2}},
		Discards: []*gameLogDiscard{
			{PlayerID: 1, Discarded: []*deck.Card{}},
			{PlayerID: 2, Discarded: []*deck.Card{}},
		},
		Result: &Result{Winners: []*Player{NewPlayer(2)}, OldPot: 400},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	assert.Equal(t, 3, hand.Rounds, "two parents plus the final hand")
	assert.Len(t, hand.Participants, 2, "parent participation is not merged in")
	assert.True(t, hand.Participant(2).Won)
}

func TestParseGameLog_Invalid(t *testing.T) {
	hand, err := ParseGameLog(json.RawMessage(`{"tricks":42}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode bourre log")
}
