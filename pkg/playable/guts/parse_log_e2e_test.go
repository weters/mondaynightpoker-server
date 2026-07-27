package guts

import (
	"encoding/json"
	"testing"
	"time"

	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/gamelog"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseGameLog_EndToEnd drives a real game through the engine and parses the
// log it actually persists.
//
// The other parser tests build the log struct directly, which pins the encoding
// but assumes the game populates those fields during play. This one closes that
// gap: it deals through Game.Deal, submits real decisions via Game.Action, takes
// the payload from GetEndOfGameDetails exactly as EndGame would, and marshals it
// the way the jsonb column stores it. If the engine ever stops recording a
// decision or a showdown, this fails even though the round-trip tests would not.
func TestParseGameLog_EndToEnd(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())
	require.NoError(t, err)
	require.NoError(t, g.Deal())

	// Round one is contested by players 1 and 3; player 2 steps aside. A contested
	// round leaves the pot to carry, so every later round is taken uncontested by
	// player 1, which ends the game.
	round := 0
	decide := func() {
		round++
		for _, p := range g.participants {
			if !g.pendingDecisions[p.PlayerID] {
				continue
			}

			in := p.PlayerID == 1 || (round == 1 && p.PlayerID == 3)
			_, _, err := g.Action(p.PlayerID, &playable.PayloadIn{
				Action:         "decide",
				AdditionalData: playable.AdditionalData{"in": in},
			})
			require.NoError(t, err, "player %d, round %d", p.PlayerID, round)
		}
	}

	// Guts settles each round on a timer. Pull the pending action's deadline into
	// the past so the game runs forward without the test sleeping.
	for i := 0; i < 50 && !g.done; i++ {
		if g.phase == PhaseDeclaration && len(g.pendingDecisions) > 0 {
			decide()
		}
		if g.pendingDealerAction != nil {
			g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
		}
		if _, err := g.Tick(); err != nil {
			require.NoError(t, err)
		}
	}

	details, over := g.GetEndOfGameDetails()
	require.True(t, over, "the game should have finished")
	require.NotNil(t, details)

	// Persist and re-read exactly as EndGame and the analytics path do.
	raw, err := json.Marshal(details.Log)
	require.NoError(t, err)

	hand, err := ParseGameLog(raw)
	require.NoError(t, err)

	assert.Equal(t, DefaultOptions().Ante, hand.AnteCents)
	assert.Len(t, hand.Participants, 3)
	assert.GreaterOrEqual(t, hand.Rounds, 1)

	// The decisions the engine recorded must survive into the normalized hand.
	assert.True(t, hand.Participant(1).VoluntarilyPlayed)
	assert.True(t, hand.Participant(3).VoluntarilyPlayed)

	// Player 2 declared out in every round, so they folded the game and the engine
	// recorded one drop-out per round they were dealt into.
	out := hand.Participant(2)
	assert.False(t, out.VoluntarilyPlayed, "declared out")
	assert.True(t, out.Folded)
	assert.Equal(t, hand.Rounds, out.Counts[gamelog.KindDropOut])

	// Two players went in, so the round was contested and somebody won it.
	assert.True(t, hand.Participant(1).WentToShowdown)
	assert.True(t, hand.Participant(3).WentToShowdown)

	won := 0
	for _, p := range hand.Participants {
		if p.Won {
			won++
		}
	}
	assert.NotZero(t, won, "a contested round has a winner")

	// Every player who was dealt in should have their starting cards recorded.
	for _, id := range []int64{1, 2, 3} {
		assert.NotEmpty(t, hand.Participant(id).StartingCards, "player %d", id)
	}
}
