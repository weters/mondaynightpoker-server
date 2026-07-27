package littlel

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
		Ante:        25,
		InitialDeal: 3,
		TradeIns:    []int{0, 1, 2},
		Community:   []*deck.Card{deck.CardFromString("14s"), deck.CardFromString("7d")},
		Seats: []*gameLogSeat{
			{PlayerID: 1, TableStake: 1000, StartingHand: []*deck.Card{deck.CardFromString("13s"), deck.CardFromString("13d")}},
			{PlayerID: 2, TableStake: 1000, StartingHand: []*deck.Card{deck.CardFromString("3c"), deck.CardFromString("8h")}},
		},
		Actions: []*gameLogAction{
			{Round: 1, PlayerID: 1, Action: action.Bet, Amount: 50},
			{Round: 1, PlayerID: 2, Action: action.Call, Amount: 50},
			{Round: 2, PlayerID: 1, Action: action.Trade, Cards: []*deck.Card{deck.CardFromString("13d")}},
			{Round: 3, PlayerID: 1, Action: action.Check},
			{Round: 3, PlayerID: 2, Action: action.Check},
		},
		Participants: []*participantJSON{
			{PlayerID: 1, Hand: deck.Hand{deck.CardFromString("13s")}},
			{PlayerID: 2, Hand: deck.Hand{deck.CardFromString("3c")}},
		},
		Winners: map[int64]int{1: 150},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	assert.Equal(t, 25, hand.AnteCents)
	assert.Len(t, hand.Board, 2)
	assert.Len(t, hand.Actions, 5)

	// Little L records an integer round rather than a named street.
	assert.Equal(t, "round-1", hand.Actions[0].Street)
	assert.Equal(t, "round-3", hand.Actions[3].Street)

	// The pot is not stored on the log, so it is reconstructed from the payouts.
	assert.Equal(t, 150, hand.PotCents)

	p1 := hand.Participant(1)
	assert.True(t, p1.VoluntarilyPlayed)
	assert.True(t, p1.WentToShowdown)
	assert.True(t, p1.Won)
	assert.Equal(t, 1, p1.Counts[gamelog.KindTrade])

	p2 := hand.Participant(2)
	assert.True(t, p2.VoluntarilyPlayed)
	assert.True(t, p2.WentToShowdown)
	assert.False(t, p2.Won)
}

func TestParseGameLog_Folded(t *testing.T) {
	log := &gameLog{
		Ante:  25,
		Seats: []*gameLogSeat{{PlayerID: 1}, {PlayerID: 2}},
		Actions: []*gameLogAction{
			{Round: 1, PlayerID: 1, Action: action.Bet, Amount: 50},
			{Round: 1, PlayerID: 2, Action: action.Fold},
		},
		Participants: []*participantJSON{
			{PlayerID: 1},
			{PlayerID: 2, DidFold: true},
		},
		Winners: map[int64]int{1: 100},
	}

	hand, err := ParseGameLog(marshalLog(t, log))
	require.NoError(t, err)

	p1 := hand.Participant(1)
	assert.True(t, p1.Won)
	assert.False(t, p1.WentToShowdown)

	p2 := hand.Participant(2)
	assert.True(t, p2.Folded)
	assert.False(t, p2.VoluntarilyPlayed)
}

func TestParseGameLog_Invalid(t *testing.T) {
	hand, err := ParseGameLog(json.RawMessage(`{"actions":true}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode little l log")
}
