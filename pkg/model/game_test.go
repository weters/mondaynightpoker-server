package model

import (
	"database/sql"
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

func TestGameRepo_GetGameByID(t *testing.T) {
	a := assert.New(t)

	_, tbl := playerAndTable()
	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	require.NoError(t, err)

	logValue := map[string]interface{}{"winner": "p1"}
	require.NoError(t, testRepos.Games.EndGame(cbg, game, logValue, map[int64]int{}))

	got, err := testRepos.Games.GetGameByID(cbg, game.ID)
	a.NoError(err)
	a.Equal(game.ID, got.ID)
	a.Equal(tbl.UUID, got.TableUUID)
	a.Equal("bourre", got.GameType)
	a.False(got.Ended.IsZero())

	// Data() exposes the jsonb log
	data, ok := got.Data().(map[string]interface{})
	if a.True(ok) {
		a.Equal("p1", data["winner"])
	}

	// missing id returns sql.ErrNoRows
	_, err = testRepos.Games.GetGameByID(cbg, 0)
	a.Equal(sql.ErrNoRows, err)
}

func TestGameRepo_ListGamesByTable(t *testing.T) {
	a := assert.New(t)

	_, tbl := playerAndTable()

	g1, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	require.NoError(t, err)
	g2, err := testRepos.Games.CreateGame(cbg, tbl, "seven-card")
	require.NoError(t, err)
	g3, err := testRepos.Games.CreateGame(cbg, tbl, "little-l")
	require.NoError(t, err)

	games, err := testRepos.Games.ListGamesByTable(cbg, tbl, 0, 100)
	a.NoError(err)
	a.Equal(3, len(games))

	// newest first (created DESC, id DESC); all created "now" so id DESC decides
	a.Equal(g3.ID, games[0].ID)
	a.Equal(g2.ID, games[1].ID)
	a.Equal(g1.ID, games[2].ID)

	// the data column is omitted from list results
	a.Nil(games[0].Data())

	// pagination
	page, err := testRepos.Games.ListGamesByTable(cbg, tbl, 1, 1)
	a.NoError(err)
	a.Equal(1, len(page))
	a.Equal(g2.ID, page[0].ID)

	count, err := testRepos.Tables.GetGamesCount(cbg, tbl)
	a.NoError(err)
	a.Equal(int64(3), count)

	// the no-data single-game fetch matches the list behavior: no log payload
	noData, err := testRepos.Games.GetGameByIDNoData(cbg, g1.ID)
	a.NoError(err)
	a.Equal(g1.ID, noData.ID)
	a.Nil(noData.Data())
}

func TestGameRepo_GetGameAdjustments(t *testing.T) {
	a := assert.New(t)

	p1, tbl := playerAndTable()
	p2 := player()
	_, err := testRepos.Tables.Join(cbg, p2, tbl)
	require.NoError(t, err)

	game1, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	require.NoError(t, err)
	require.NoError(t, testRepos.Games.EndGame(cbg, game1, nil, map[int64]int{p1.ID: 100, p2.ID: -100}))

	game2, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	require.NoError(t, err)
	require.NoError(t, testRepos.Games.EndGame(cbg, game2, nil, map[int64]int{p1.ID: -40, p2.ID: 40}))

	adjustments, err := testRepos.Games.GetGameAdjustments(cbg, []int64{game1.ID, game2.ID})
	a.NoError(err)

	// game1: p1 wins 100, p2 loses 100 — ordered by adjustment DESC
	g1adj := adjustments[game1.ID]
	if a.Equal(2, len(g1adj)) {
		a.Equal(p1.ID, g1adj[0].PlayerID)
		a.Equal(100, g1adj[0].Adjustment)
		a.Equal(p2.ID, g1adj[1].PlayerID)
		a.Equal(-100, g1adj[1].Adjustment)
		a.NotEmpty(g1adj[0].DisplayName)
	}

	// game2: p2 wins 40, p1 loses 40
	g2adj := adjustments[game2.ID]
	if a.Equal(2, len(g2adj)) {
		a.Equal(p2.ID, g2adj[0].PlayerID)
		a.Equal(40, g2adj[0].Adjustment)
		a.Equal(p1.ID, g2adj[1].PlayerID)
		a.Equal(-40, g2adj[1].Adjustment)
	}

	// a game with no ledger rows is absent from the map
	empty, err := testRepos.Games.GetGameAdjustments(cbg, []int64{})
	a.NoError(err)
	a.Equal(0, len(empty))
}
