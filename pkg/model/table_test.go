package model

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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
	_, _ = playerAndTable()
	p2, tbl2 := playerAndTable()
	p3, tbl3 := playerAndTable()
	_, _ = playerAndTable()

	a := assert.New(t)
	tables, err := testRepos.Tables.GetTables(cbg, 1, 2)
	a.NoError(err)
	a.Equal(2, len(tables))
	a.Equal(p3.Email, tables[0].Email)
	a.Equal(tbl3.UUID, tables[0].UUID)
	a.Equal(p2.Email, tables[1].Email)
	a.Equal(tbl2.UUID, tables[1].UUID)

	// sanity check
	a.NotEqual(p2.Email, p3.Email)
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
