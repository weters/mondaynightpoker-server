package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGameRepo_EndGame(t *testing.T) {
	a := assert.New(t)

	p1, tbl := playerAndTable()
	p2 := player()
	_, err := testRepos.Tables.Join(cbg, p2, tbl)
	require.NoError(t, err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	require.NoError(t, err)
	a.True(game.Ended.IsZero())

	logValue := map[string]interface{}{"winner": "p1"}
	require.NoError(t, testRepos.Games.EndGame(cbg, game, logValue, map[int64]int{
		p1.ID: 100,
		p2.ID: -100,
	}))
	a.False(game.Ended.IsZero())

	// balances are adjusted for every player at the table
	players, err := testRepos.Tables.GetPlayers(cbg, tbl)
	require.NoError(t, err)
	require.Equal(t, 2, len(players))

	balances := make(map[int64]int)
	for _, pt := range players {
		balances[pt.PlayerID] = pt.Balance
	}
	a.Equal(100, balances[p1.ID])
	a.Equal(-100, balances[p2.ID])

	// the games row is marked as ended
	var endedIsSet bool
	row := testDB.QueryRow("SELECT ended IS NOT NULL FROM games WHERE id = $1", game.ID)
	require.NoError(t, row.Scan(&endedIsSet))
	a.True(endedIsSet)
}
