package mcpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/model"
)

func TestFromCards(t *testing.T) {
	a := assert.New(t)

	got := fromCards([]*deck.Card{
		{Rank: 14, Suit: deck.Spades},
		nil, // a card that was never dealt or is not visible to the caller
		{Rank: 11, Suit: deck.Hearts, IsWild: true},
	})

	require.Len(t, got, 2, "nil cards are skipped")
	a.Equal("A♠", got[0].Display)
	a.False(got[0].IsWild)
	a.Equal("J♡", got[1].Display)
	a.True(got[1].IsWild)
}

// TestFromCards_UnknownSuit pins that a card the renderer cannot describe loses its
// display string rather than panicking. These cards come out of persisted jsonb, so
// nothing guarantees the suit is one deck.Card.String knows.
func TestFromCards_UnknownSuit(t *testing.T) {
	a := assert.New(t)

	var got []CardDTO
	a.NotPanics(func() {
		got = fromCards([]*deck.Card{{Rank: 0, Suit: deck.Suit("")}, {Rank: 5, Suit: deck.Suit("swords")}})
	})

	require.Len(t, got, 2)
	for _, c := range got {
		a.Empty(c.Display, "an unrenderable card has no display string")
	}
	a.Equal("swords", got[1].Suit, "the raw suit is still reported")
}

func TestNewRate(t *testing.T) {
	a := assert.New(t)

	got := newRate(0.625, true)
	require.NotNil(t, got)
	a.Equal(0.625, got.Value)
	a.Equal("62.5%", got.Display)

	// An undefined statistic is absent rather than zero: "0.0%" would read as a
	// real measurement.
	a.Nil(newRate(0, false))
}

func TestRoundCents(t *testing.T) {
	for in, want := range map[float64]int{
		0:      0,
		12.4:   12,
		12.5:   13,
		12.6:   13,
		-12.4:  -12,
		-12.5:  -13,
		-12.6:  -13,
		100.49: 100,
	} {
		assert.Equal(t, want, roundCents(in), "roundCents(%v)", in)
	}
}

func TestFromSpread(t *testing.T) {
	a := assert.New(t)

	got := fromSpread(model.NewSpread([]int{100, -50, 25}))

	a.Equal(3, got.Count)
	a.Equal(75, got.TotalCents)
	a.Equal("$0.75", got.TotalDisplay)
	a.InDelta(25.0, got.MeanCents, 0.001)
	a.Equal("$0.25", got.MeanDisplay)
	a.Equal("$0.25", got.MedianDisplay)
	a.Equal("$1", got.BestDisplay)
	a.Equal("-$0.50", got.WorstDisplay)
}

func TestFromStreaks(t *testing.T) {
	a := assert.New(t)

	got := fromStreaks(model.ComputeStreaks([]model.StreakInput{
		{NetCents: 100}, {NetCents: 50}, {NetCents: -20},
	}))

	require.NotNil(t, got.LongestWinning)
	a.Equal("winning", got.LongestWinning.Outcome)
	a.Equal(2, got.LongestWinning.Length)
	a.Equal("$1.50", got.LongestWinning.NetDisplay)

	require.NotNil(t, got.LongestLosing)
	a.Equal(1, got.LongestLosing.Length)

	require.NotNil(t, got.Current)
	a.Equal("losing", got.Current.Outcome)
}

// TestFromStreaks_Absent pins that a run that never happened is omitted rather than
// reported as a zero-length streak.
func TestFromStreaks_Absent(t *testing.T) {
	a := assert.New(t)

	got := fromStreaks(model.ComputeStreaks([]model.StreakInput{{NetCents: 100}}))

	a.NotNil(got.LongestWinning)
	a.Nil(got.LongestLosing, "never lost, so there is no losing streak")
	a.NotNil(got.Current)

	a.Nil(fromStreaks(model.ComputeStreaks(nil)).Current)
}
