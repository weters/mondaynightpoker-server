package gamefactory

import (
	"sort"
	"testing"

	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPlayerTables builds PlayerTable fixtures for factory tests
func testPlayerTables(ids ...int64) []*model.PlayerTable {
	players := make([]*model.PlayerTable, len(ids))
	for i, id := range ids {
		players[i] = &model.PlayerTable{
			Player:     &model.Player{ID: id},
			PlayerID:   id,
			TableStake: 10000,
			Active:     true,
		}
	}

	return players
}

// Test_factorySlugsMatchPlayerState ensures the slug each game reports in its player
// state matches the key it is registered under. The frontend starts a game by sending
// the registry key as the createGame subject and picks the component to render from
// the player-state value, so a mismatch renders a blank screen client-side.
func Test_factorySlugsMatchPlayerState(t *testing.T) {
	createArgs := map[string]struct {
		playerIDs []int64
		data      playable.AdditionalData
	}{
		"bourre":        {[]int64{1, 2, 3}, playable.AdditionalData{"ante": float64(25)}},
		"seven-card":    {[]int64{1, 2, 3}, playable.AdditionalData{"variant": "stud", "ante": float64(25)}},
		"pass-the-poop": {[]int64{1, 2, 3}, playable.AdditionalData{"edition": "standard", "ante": float64(25)}},
		"little-l":      {[]int64{1, 2, 3}, playable.AdditionalData{"tradeIns": []float64{0, 2}, "ante": float64(25)}},
		"acey-deucey":   {[]int64{1, 2}, playable.AdditionalData{}},
		"texas-hold-em": {[]int64{1, 2, 3}, playable.AdditionalData{}},
		"guts":          {[]int64{1, 2}, playable.AdditionalData{"ante": float64(25)}},
	}

	assert.Len(t, createArgs, len(factories), "createArgs must exactly cover the registered factories")

	for slug, factory := range factories {
		t.Run(slug, func(t *testing.T) {
			args, ok := createArgs[slug]
			require.True(t, ok, "add createGame args for new game %q", slug)

			game, err := factory.CreateGame(logrus.StandardLogger(), testPlayerTables(args.playerIDs...), args.data)
			require.NoError(t, err)

			state, err := game.GetPlayerState(args.playerIDs[0])
			require.NoError(t, err)

			assert.Equal(t, "game", state.Key)
			assert.Equal(t, slug, state.Value, "player-state value must match the factory registry key")
		})
	}
}

// Test_Names ensures Names returns exactly the registered slugs, sorted.
func Test_Names(t *testing.T) {
	expected := make([]string, 0, len(factories))
	for slug := range factories {
		expected = append(expected, slug)
	}
	sort.Strings(expected)

	names := Names()
	assert.Equal(t, expected, names)
	assert.True(t, sort.StringsAreSorted(names), "Names() must be sorted")
}
