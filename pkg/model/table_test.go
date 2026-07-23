package model

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setTableCreated overrides a table's created timestamp so date-filtered queries
// can be tested in an isolated window that no other table (created via now())
// lands in. The window is derived from a unique per-run instant.
func setTableCreated(t *testing.T, tableUUID string, created time.Time) {
	t.Helper()
	_, err := testDB.Exec(`UPDATE tables SET created = $1 WHERE uuid = $2`, created, tableUUID)
	require.NoError(t, err)
}

func TestTable_CreateGame(t *testing.T) {
	_, tbl := playerAndTable()
	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	assert.NoError(t, err)
	assert.NotNil(t, game)
}

func TestGetTableByUUID(t *testing.T) {
	tbl, err := testRepos.Tables.GetTableByUUID(cbg, uuid.New().String())
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, tbl)

	_, tbl2 := playerAndTable()
	tbl, err = testRepos.Tables.GetTableByUUID(cbg, strings.ToLower(tbl2.UUID))
	assert.NoError(t, err)
	assert.Equal(t, tbl.Name, tbl2.Name)

	// check to see if UUID is case-insensitive
	tbl, err = testRepos.Tables.GetTableByUUID(cbg, strings.ToUpper(tbl2.UUID))
	assert.NoError(t, err)
	assert.Equal(t, tbl.Name, tbl2.Name)
}

func playerAndTable() (*Player, *Table) {
	p := player()
	t, err := testRepos.Tables.CreateTable(cbg, p, "test table")
	if err != nil {
		panic(err)
	}

	return p, t
}

func TestTable_GetGamesCount(t *testing.T) {
	_, tbl := playerAndTable()

	c, err := testRepos.Tables.GetGamesCount(cbg, tbl)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), c)

	_, _ = testRepos.Games.CreateGame(cbg, tbl, "bourre")

	c, err = testRepos.Tables.GetGamesCount(cbg, tbl)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), c)
}

func TestTable_GetActivePlayersShifted(t *testing.T) {
	p0, tbl := playerAndTable()
	p1 := player()
	p2 := player()
	p3 := player()
	p4 := player()

	_, _ = testRepos.Tables.Join(cbg, p1, tbl)
	_, _ = testRepos.Tables.Join(cbg, p2, tbl)
	_, _ = testRepos.Tables.Join(cbg, p3, tbl)
	pt4, _ := testRepos.Tables.Join(cbg, p4, tbl)

	pt4.Active = false
	_ = testRepos.Tables.SavePlayerTable(cbg, pt4)

	players, err := testRepos.Tables.GetActivePlayersShifted(cbg, tbl)
	assert.NoError(t, err)
	assert.Equal(t, p0.ID, players[0].PlayerID)
	assert.Equal(t, p1.ID, players[1].PlayerID)
	assert.Equal(t, p2.ID, players[2].PlayerID)
	assert.Equal(t, p3.ID, players[3].PlayerID)

	_, _ = testRepos.Games.CreateGame(cbg, tbl, "bourre")
	players, err = testRepos.Tables.GetActivePlayersShifted(cbg, tbl)
	assert.NoError(t, err)
	assert.Equal(t, p1.ID, players[0].PlayerID)
	assert.Equal(t, p2.ID, players[1].PlayerID)
	assert.Equal(t, p3.ID, players[2].PlayerID)
	assert.Equal(t, p0.ID, players[3].PlayerID)

	_, _ = testRepos.Games.CreateGame(cbg, tbl, "bourre")
	players, err = testRepos.Tables.GetActivePlayersShifted(cbg, tbl)
	assert.NoError(t, err)
	assert.Equal(t, p2.ID, players[0].PlayerID)
	assert.Equal(t, p3.ID, players[1].PlayerID)
	assert.Equal(t, p0.ID, players[2].PlayerID)
	assert.Equal(t, p1.ID, players[3].PlayerID)

	_, _ = testRepos.Games.CreateGame(cbg, tbl, "bourre")
	players, err = testRepos.Tables.GetActivePlayersShifted(cbg, tbl)
	assert.NoError(t, err)
	assert.Equal(t, p3.ID, players[0].PlayerID)
	assert.Equal(t, p0.ID, players[1].PlayerID)
	assert.Equal(t, p1.ID, players[2].PlayerID)
	assert.Equal(t, p2.ID, players[3].PlayerID)

	_, _ = testRepos.Games.CreateGame(cbg, tbl, "bourre")
	_, _ = testRepos.Games.CreateGame(cbg, tbl, "bourre")
	players, err = testRepos.Tables.GetActivePlayersShifted(cbg, tbl)
	assert.NoError(t, err)
	assert.Equal(t, p1.ID, players[0].PlayerID)
	assert.Equal(t, p2.ID, players[1].PlayerID)
	assert.Equal(t, p3.ID, players[2].PlayerID)
	assert.Equal(t, p0.ID, players[3].PlayerID)
}

func TestTable_GetActivePlayersShifted_noActivePlayers(t *testing.T) {
	p0, tbl := playerAndTable()
	p1 := player()

	pt0, _ := testRepos.Tables.GetPlayerTable(cbg, p0, tbl)
	pt1, _ := testRepos.Tables.Join(cbg, p1, tbl)

	pt0.Active = false
	_ = testRepos.Tables.SavePlayerTable(cbg, pt0)

	pt1.Active = false
	_ = testRepos.Tables.SavePlayerTable(cbg, pt1)

	players, err := testRepos.Tables.GetActivePlayersShifted(cbg, tbl)
	assert.NoError(t, err)
	assert.Equal(t, []*PlayerTable{}, players)
}

func TestGetTables(t *testing.T) {
	a := assert.New(t)

	type playerTable struct {
		player *Player
		table  *Table
	}

	// each playerAndTable() call creates a table under a fresh, unique-email player,
	// so these can be picked back out even if other tables are created concurrently
	var entries []playerTable
	for i := 0; i < 4; i++ {
		p, tbl := playerAndTable()
		entries = append(entries, playerTable{player: p, table: tbl})
	}

	// other test packages running concurrently against the same database may create
	// tables of their own, so a small page cannot be asserted positionally. Just
	// confirm pagination is honored by page size...
	tables, err := testRepos.Tables.GetTables(cbg, 1, 2)
	a.NoError(err)
	a.Equal(2, len(tables))

	// ...then fetch a generously large page and filter down to this test's own
	// tables (identified by the fresh, unique player emails) to verify pagination
	// correctness, the email join, field mapping, and ordering.
	all, err := testRepos.Tables.GetTables(cbg, 0, 1000)
	a.NoError(err)

	emails := make(map[string]bool, len(entries))
	for _, e := range entries {
		emails[e.player.Email] = true
	}

	var mine []*TableWithPlayerEmail
	for _, tbl := range all {
		if emails[tbl.Email] {
			mine = append(mine, tbl)
		}
	}

	// GetTables orders newest first, so mine[0] should be the most recently created entry
	if a.Equal(len(entries), len(mine)) {
		for i, tbl := range mine {
			expected := entries[len(entries)-1-i]
			a.Equal(expected.player.Email, tbl.Email)
			a.Equal(expected.table.UUID, tbl.UUID)
		}
	}

	// sanity check
	a.NotEqual(entries[0].player.Email, entries[1].player.Email)
}

func TestTableRepo_CloneTable_randomizesOrder(t *testing.T) {
	a := assert.New(t)

	admin, source := playerAndTable()
	admin.IsSiteAdmin = true
	a.NoError(testRepos.Players.Save(cbg, admin))

	for i := 0; i < 7; i++ {
		_, _ = testRepos.Tables.Join(cbg, player(), source)
	}

	srcPlayers, _ := testRepos.Tables.GetPlayers(cbg, source)
	srcOrder := make([]int64, len(srcPlayers))
	for i, sp := range srcPlayers {
		srcOrder[i] = sp.PlayerID
	}

	// Across several clones, ordering should differ from the source at least once.
	differs := false
	for i := 0; i < 8 && !differs; i++ {
		cloned, err := testRepos.Tables.CloneTable(cbg, admin, source, fmt.Sprintf("clone-%d", i))
		a.NoError(err)

		clonedPlayers, _ := testRepos.Tables.GetPlayers(cbg, cloned)
		a.Equal(len(srcOrder), len(clonedPlayers))
		for j, cp := range clonedPlayers {
			if cp.PlayerID != srcOrder[j] {
				differs = true
				break
			}
		}
	}
	a.True(differs, "expected at least one cloned ordering to differ from the source order")
}

func TestTableRepo_CloneTable_carriesOverPlayers(t *testing.T) {
	a := assert.New(t)

	_, source := playerAndTable()

	siteAdmin := player()
	siteAdmin.IsSiteAdmin = true
	a.NoError(testRepos.Players.Save(cbg, siteAdmin))

	cloned, err := testRepos.Tables.CloneTable(cbg, siteAdmin, source, "Clone")
	a.NoError(err)
	a.NotNil(cloned)
	a.Equal("Clone", cloned.Name)
	a.Equal(siteAdmin.ID, cloned.PlayerID)

	srcPlayers, _ := testRepos.Tables.GetPlayers(cbg, source)
	clonedPlayers, _ := testRepos.Tables.GetPlayers(cbg, cloned)
	a.Equal(len(srcPlayers), len(clonedPlayers))

	// cloned players sit out with zeroed balances
	for _, cp := range clonedPlayers {
		a.Equal(0, cp.Balance)
		a.False(cp.Active)
	}
}

func TestGetTableStats(t *testing.T) {
	a := assert.New(t)

	p1, tbl := playerAndTable()

	// p1 plays two games
	g1, _ := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, g1, nil, map[int64]int{p1.ID: 100}))
	g2, _ := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, g2, nil, map[int64]int{p1.ID: -30}))

	// p2 joins after the games, so has a roster row but no game transactions
	p2 := player()
	_, err := testRepos.Tables.Join(cbg, p2, tbl)
	require.NoError(t, err)

	stats, err := testRepos.Tables.GetTableStats(cbg, tbl)
	a.NoError(err)
	require.Equal(t, 2, len(stats))

	// ordered by net winnings DESC: p1 (70) then p2 (0)
	a.Equal(p1.ID, stats[0].PlayerID)
	a.Equal(70, stats[0].NetWinnings)
	a.Equal(70, stats[0].Balance)
	a.Equal(2, stats[0].GamesPlayed)
	a.NotEmpty(stats[0].DisplayName)

	// p2 appears with zeros (LEFT JOIN)
	a.Equal(p2.ID, stats[1].PlayerID)
	a.Equal(0, stats[1].NetWinnings)
	a.Equal(0, stats[1].Balance)
	a.Equal(0, stats[1].GamesPlayed)
}

func TestGetTableAggregates(t *testing.T) {
	a := assert.New(t)

	p1, tbl := playerAndTable()
	p2 := player()
	_, err := testRepos.Tables.Join(cbg, p2, tbl)
	require.NoError(t, err)

	// empty table: 2 players, 0 games, 0 balance
	agg, err := testRepos.Tables.GetTableAggregates(cbg, tbl)
	a.NoError(err)
	a.Equal(2, agg.PlayersCount)
	a.Equal(int64(0), agg.GamesCount)
	a.Equal(0, agg.TotalBalance)

	g1, _ := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, g1, nil, map[int64]int{p1.ID: 100, p2.ID: -100}))
	g2, _ := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, g2, nil, map[int64]int{p1.ID: 40, p2.ID: -40}))

	agg, err = testRepos.Tables.GetTableAggregates(cbg, tbl)
	a.NoError(err)
	a.Equal(2, agg.PlayersCount)
	a.Equal(int64(2), agg.GamesCount)
	// balances net to zero across the two players
	a.Equal(0, agg.TotalBalance)
}

func TestGetActiveTablesFiltered(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true // to rapidly create tables
	live1, _ := testRepos.Tables.CreateTable(cbg, p, "Filtered Live 1")
	live2, _ := testRepos.Tables.CreateTable(cbg, p, "Filtered Live 2")
	gone, _ := testRepos.Tables.CreateTable(cbg, p, "Filtered Deleted")

	// GetActiveTablesFiltered is global (not player-scoped), so isolate this test's
	// tables into a tight, unique created window that no other table lands in. The
	// base is a far-future instant offset by a per-run nanosecond value; the window
	// spans only a couple of milliseconds so a separate run seconds later cannot
	// collide with it.
	base := time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(time.Now().UnixNano()))
	setTableCreated(t, live1.UUID, base)
	setTableCreated(t, live2.UUID, base.Add(time.Millisecond))
	setTableCreated(t, gone.UUID, base.Add(2*time.Millisecond))

	gone.Deleted = true
	require.NoError(t, testRepos.Tables.Save(cbg, gone))

	from := base
	to := base.Add(2 * time.Millisecond)

	tables, err := testRepos.Tables.GetActiveTablesFiltered(cbg, from, to, 0, 100)
	a.NoError(err)
	// only the two live tables, newest (created DESC) first
	require.Equal(t, 2, len(tables))
	a.Equal(live2.UUID, tables[0].UUID)
	a.Equal(live1.UUID, tables[1].UUID)
	a.Equal(p.Email, tables[0].Email)

	count, err := testRepos.Tables.GetActiveTablesCount(cbg, from, to)
	a.NoError(err)
	a.Equal(int64(2), count)

	// pagination is honored
	page, err := testRepos.Tables.GetActiveTablesFiltered(cbg, from, to, 1, 1)
	a.NoError(err)
	require.Equal(t, 1, len(page))
	a.Equal(live1.UUID, page[0].UUID)

	// a distinct far window that no table occupies excludes everything
	lo := time.Date(2150, 1, 1, 0, 0, 0, 0, time.UTC)
	hi := time.Date(2150, 1, 2, 0, 0, 0, 0, time.UTC)
	empty, err := testRepos.Tables.GetActiveTablesFiltered(cbg, lo, hi, 0, 100)
	a.NoError(err)
	a.Equal(0, len(empty))
	emptyCount, err := testRepos.Tables.GetActiveTablesCount(cbg, lo, hi)
	a.NoError(err)
	a.Equal(int64(0), emptyCount)
}

func TestTable_Save(t *testing.T) {
	a := assert.New(t)

	_, table := playerAndTable()
	a.False(table.Deleted)

	modifiedBefore := table.Modified

	origName := table.Name
	table.Name = origName + "-updated"
	table.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, table))

	table, err := testRepos.Tables.GetTableByUUID(cbg, table.UUID)
	a.NoError(err)
	a.NotEqual(origName, table.Name)
	a.True(table.Deleted)
	a.False(table.Modified.Before(modifiedBefore), "modified should not be before the original modified time")
}
