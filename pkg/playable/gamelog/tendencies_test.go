package gamelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTendencies_Add(t *testing.T) {
	var got Tendencies

	got.Add(&Participation{
		PlayerID:           1,
		VoluntarilyPlayed:  true,
		WentToShowdown:     true,
		Won:                true,
		AmountWageredCents: 250,
		Counts:             map[Kind]int{KindBet: 1, KindRaise: 1, KindCall: 2, KindCheck: 3},
	})
	got.Add(&Participation{
		PlayerID: 1,
		Folded:   true,
		Counts:   map[Kind]int{KindFold: 1},
	})
	got.Add(&Participation{
		PlayerID:          1,
		VoluntarilyPlayed: true,
		WentToShowdown:    true,
		Counts:            map[Kind]int{KindCall: 1, KindDropOut: 1},
	})

	assert.Equal(t, 3, got.HandsPlayed)
	assert.Equal(t, 2, got.HandsVoluntarilyPlayed)
	assert.Equal(t, 1, got.HandsFolded)
	assert.Equal(t, 2, got.HandsToShowdown)
	assert.Equal(t, 1, got.HandsWonAtShowdown)
	assert.Equal(t, 1, got.HandsWon)
	assert.Equal(t, 250, got.AmountWageredCents)

	assert.Equal(t, 1, got.Bets)
	assert.Equal(t, 1, got.Raises)
	assert.Equal(t, 3, got.Calls)
	assert.Equal(t, 3, got.Checks)
	assert.Equal(t, 2, got.Folds, "drop-outs count as folds")
}

func TestTendencies_Add_Nil(t *testing.T) {
	var got Tendencies
	got.Add(nil)

	assert.Zero(t, got.HandsPlayed)
}

func TestTendencies_AggressionFactor(t *testing.T) {
	got := Tendencies{Bets: 3, Raises: 3, Calls: 2}
	factor, ok := got.AggressionFactor()
	assert.True(t, ok)
	assert.Equal(t, 3.0, factor)

	// A player who has never called has no defined aggression factor: reporting
	// one would divide by zero and imply infinite aggression from a single bet.
	never := Tendencies{Bets: 5}
	factor, ok = never.AggressionFactor()
	assert.False(t, ok)
	assert.Zero(t, factor)
}

func TestTendencies_Rate(t *testing.T) {
	got := Tendencies{HandsPlayed: 4, HandsVoluntarilyPlayed: 1}

	rate, ok := got.Rate(got.HandsVoluntarilyPlayed)
	assert.True(t, ok)
	assert.Equal(t, 0.25, rate)

	var empty Tendencies
	rate, ok = empty.Rate(0)
	assert.False(t, ok)
	assert.Zero(t, rate)
}

func TestTendencies_ShowdownWinRate(t *testing.T) {
	got := Tendencies{HandsPlayed: 10, HandsToShowdown: 4, HandsWonAtShowdown: 3}

	// The denominator is showdowns reached, not hands played: folding more often
	// does not make someone better at showdowns.
	rate, ok := got.ShowdownWinRate()
	assert.True(t, ok)
	assert.Equal(t, 0.75, rate)

	nitty := Tendencies{HandsPlayed: 50}
	rate, ok = nitty.ShowdownWinRate()
	assert.False(t, ok)
	assert.Zero(t, rate)
}
