package texasholdem

import (
	"encoding/json"
	"testing"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"
	"mondaynightpoker-server/pkg/playable/poker/action"

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
		Variant:    Pineapple,
		Ante:       0,
		SmallBlind: 25,
		BigBlind:   50,
		Pot:        300,
		Community:  deck.Hand{deck.CardFromString("14s"), deck.CardFromString("7d"), deck.CardFromString("2c")},
		Seats: []*gameLogSeat{
			{PlayerID: 1, TableStake: 1000, HoleCards: []*deck.Card{deck.CardFromString("13s"), deck.CardFromString("13d")}},
			{PlayerID: 2, TableStake: 1000, HoleCards: []*deck.Card{deck.CardFromString("3c"), deck.CardFromString("8h")}},
			{PlayerID: 3, TableStake: 1000, HoleCards: []*deck.Card{deck.CardFromString("9h"), deck.CardFromString("9c")}},
		},
		Actions: []*gameLogAction{
			{Street: "preflop", PlayerID: 1, Action: action.Raise, Amount: 150},
			{Street: "preflop", PlayerID: 2, Action: action.Fold},
			{Street: "preflop", PlayerID: 3, Action: action.Call, Amount: 150},
			{Street: "flop", PlayerID: 1, Action: action.Bet, Amount: 100},
			{Street: "flop", PlayerID: 3, Action: action.Call, Amount: 100},
		},
		Participants: []*participantJSON{
			{PlayerID: 1, Winnings: 300},
			{PlayerID: 2, Folded: true},
			{PlayerID: 3},
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	// The variant is written by MarshalJSON as an {"id","name"} object and has no
	// matching UnmarshalJSON, so this asserts the object form decodes.
	assert.Equal(t, "pineapple", hand.Variant)
	assert.Equal(t, 300, hand.PotCents)
	assert.Len(t, hand.Board, 3)
	assert.Len(t, hand.Actions, 5)

	// Actions are written the same way and must survive the same round trip.
	assert.Equal(t, gamelog.KindRaise, hand.Actions[0].Kind)
	assert.Equal(t, 150, hand.Actions[0].AmountCents)
	assert.Equal(t, gamelog.KindFold, hand.Actions[1].Kind)

	p1 := hand.Participant(1)
	assert.True(t, p1.VoluntarilyPlayed)
	assert.True(t, p1.WentToShowdown)
	assert.True(t, p1.Won)
	assert.Equal(t, 250, p1.AmountWageredCents)
	assert.Equal(t, 1, p1.Counts[gamelog.KindRaise])
	assert.Equal(t, 1, p1.Counts[gamelog.KindBet])

	p2 := hand.Participant(2)
	assert.False(t, p2.VoluntarilyPlayed, "folding preflop is not voluntary play")
	assert.True(t, p2.Folded)
	assert.False(t, p2.WentToShowdown)

	p3 := hand.Participant(3)
	assert.True(t, p3.VoluntarilyPlayed, "calling is voluntary")
	assert.True(t, p3.WentToShowdown)
	assert.False(t, p3.Won)
}

// TestParseGameLog_UncontestedWin covers the distinction between winning a hand
// and winning it at showdown: a pot everyone else folded out of is not a showdown.
func TestParseGameLog_UncontestedWin(t *testing.T) {
	log := &gameLog{
		Variant: Standard,
		Pot:     75,
		Seats: []*gameLogSeat{
			{PlayerID: 1, HoleCards: []*deck.Card{deck.CardFromString("14s"), deck.CardFromString("14d")}},
			{PlayerID: 2, HoleCards: []*deck.Card{deck.CardFromString("3c"), deck.CardFromString("8h")}},
		},
		Actions: []*gameLogAction{
			{Street: "preflop", PlayerID: 1, Action: action.Raise, Amount: 75},
			{Street: "preflop", PlayerID: 2, Action: action.Fold},
		},
		Participants: []*participantJSON{
			{PlayerID: 1, Winnings: 75},
			{PlayerID: 2, Folded: true},
		},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	p1 := hand.Participant(1)
	assert.True(t, p1.Won)
	assert.False(t, p1.WentToShowdown, "no opponent stayed in, so there was no showdown")
}

// TestParseGameLog_CheckIsNotVoluntary pins the VPIP rule: a player who only ever
// checks has not voluntarily put money in the pot.
func TestParseGameLog_CheckIsNotVoluntary(t *testing.T) {
	log := &gameLog{
		Variant: Standard,
		Seats:   []*gameLogSeat{{PlayerID: 1}, {PlayerID: 2}},
		Actions: []*gameLogAction{
			{Street: "flop", PlayerID: 1, Action: action.Check},
			{Street: "flop", PlayerID: 2, Action: action.Check},
		},
		Participants: []*participantJSON{{PlayerID: 1}, {PlayerID: 2}},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	assert.False(t, hand.Participant(1).VoluntarilyPlayed)
	assert.Equal(t, 1, hand.Participant(1).Counts[gamelog.KindCheck])
}

func TestParseGameLog_Invalid(t *testing.T) {
	hand, err := ParseGameLog(json.RawMessage(`{"seats":5}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode texas hold'em log")
}
