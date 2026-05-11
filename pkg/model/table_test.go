package model

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTable_CreateGame(t *testing.T) {
	_, tbl := playerAndTable()
	game, err := tbl.CreateGame(cbg, "bourre")
	assert.NoError(t, err)
	assert.NotNil(t, game)
}

func TestGetTableByUUID(t *testing.T) {
	tbl, err := GetTableByUUID(cbg, uuid.New().String())
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, tbl)

	_, tbl2 := playerAndTable()
	tbl, err = GetTableByUUID(cbg, strings.ToLower(tbl2.UUID))
	assert.NoError(t, err)
	assert.Equal(t, tbl.Name, tbl2.Name)

	// check to see if UUID is case-insensitive
	tbl, err = GetTableByUUID(cbg, strings.ToUpper(tbl2.UUID))
	assert.NoError(t, err)
	assert.Equal(t, tbl.Name, tbl2.Name)
}

func TestTable_Players(t *testing.T) {
	p1, tbl := playerAndTable()
	p2 := player()
	p3 := player()

	pt, _ := p2.Join(cbg, tbl)
	_ = pt.AdjustBalance(cbg, 10, "no reason", nil)

	_, _ = p3.Join(cbg, tbl)

	players, err := tbl.GetPlayers(cbg)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(players))

	assert.Equal(t, p1.ID, players[0].Player.ID)
	assert.Equal(t, 0, players[0].Balance)

	assert.Equal(t, p2.ID, players[1].Player.ID)
	assert.Equal(t, 10, players[1].Balance)

	assert.Equal(t, p3.ID, players[2].Player.ID)
	assert.Equal(t, 0, players[2].Balance)
}

func TestTable_Reload(t *testing.T) {
	_, tbl := playerAndTable()
	tbl2 := &Table{UUID: tbl.UUID}
	tbl2.Name = "Different"
	assert.NoError(t, tbl2.Reload(cbg))
	assert.Equal(t, "test table", tbl2.Name)
}

func playerAndTable() (*Player, *Table) {
	p := player()
	t, err := p.CreateTable(cbg, "test table")
	if err != nil {
		panic(err)
	}

	return p, t
}

func TestTable_GetGamesCount(t *testing.T) {
	_, tbl := playerAndTable()

	c, err := tbl.GetGamesCount(cbg)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), c)

	_, _ = tbl.CreateGame(cbg, "bourre")

	c, err = tbl.GetGamesCount(cbg)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), c)
}

func TestTable_GetActivePlayersShifted(t *testing.T) {
	p0, tbl := playerAndTable()
	p1 := player()
	p2 := player()
	p3 := player()
	p4 := player()

	_, _ = p1.Join(cbg, tbl)
	_, _ = p2.Join(cbg, tbl)
	_, _ = p3.Join(cbg, tbl)
	pt4, _ := p4.Join(cbg, tbl)

	pt4.Active = false
	_ = pt4.Save(cbg)

	players, err := tbl.GetActivePlayersShifted(cbg)
	assert.NoError(t, err)
	assert.Equal(t, p0.ID, players[0].PlayerID)
	assert.Equal(t, p1.ID, players[1].PlayerID)
	assert.Equal(t, p2.ID, players[2].PlayerID)
	assert.Equal(t, p3.ID, players[3].PlayerID)

	_, _ = tbl.CreateGame(cbg, "bourre")
	players, err = tbl.GetActivePlayersShifted(cbg)
	assert.NoError(t, err)
	assert.Equal(t, p1.ID, players[0].PlayerID)
	assert.Equal(t, p2.ID, players[1].PlayerID)
	assert.Equal(t, p3.ID, players[2].PlayerID)
	assert.Equal(t, p0.ID, players[3].PlayerID)

	_, _ = tbl.CreateGame(cbg, "bourre")
	players, err = tbl.GetActivePlayersShifted(cbg)
	assert.NoError(t, err)
	assert.Equal(t, p2.ID, players[0].PlayerID)
	assert.Equal(t, p3.ID, players[1].PlayerID)
	assert.Equal(t, p0.ID, players[2].PlayerID)
	assert.Equal(t, p1.ID, players[3].PlayerID)

	_, _ = tbl.CreateGame(cbg, "bourre")
	players, err = tbl.GetActivePlayersShifted(cbg)
	assert.NoError(t, err)
	assert.Equal(t, p3.ID, players[0].PlayerID)
	assert.Equal(t, p0.ID, players[1].PlayerID)
	assert.Equal(t, p1.ID, players[2].PlayerID)
	assert.Equal(t, p2.ID, players[3].PlayerID)

	_, _ = tbl.CreateGame(cbg, "bourre")
	_, _ = tbl.CreateGame(cbg, "bourre")
	players, err = tbl.GetActivePlayersShifted(cbg)
	assert.NoError(t, err)
	assert.Equal(t, p1.ID, players[0].PlayerID)
	assert.Equal(t, p2.ID, players[1].PlayerID)
	assert.Equal(t, p3.ID, players[2].PlayerID)
	assert.Equal(t, p0.ID, players[3].PlayerID)
}

func TestTable_GetActivePlayersShifted_noActivePlayers(t *testing.T) {
	p0, tbl := playerAndTable()
	p1 := player()

	pt0, _ := p0.GetPlayerTable(cbg, tbl)
	pt1, _ := p1.Join(cbg, tbl)

	pt0.Active = false
	_ = pt0.Save(cbg)

	pt1.Active = false
	_ = pt1.Save(cbg)

	players, err := tbl.GetActivePlayersShifted(cbg)
	assert.NoError(t, err)
	assert.Equal(t, []*PlayerTable{}, players)
}

func TestGetTables(t *testing.T) {
	_, _ = playerAndTable()
	p2, tbl2 := playerAndTable()
	p3, tbl3 := playerAndTable()
	_, _ = playerAndTable()

	a := assert.New(t)
	tables, err := GetTables(cbg, 1, 2)
	a.NoError(err)
	a.Equal(2, len(tables))
	a.Equal(p3.Email, tables[0].Email)
	a.Equal(tbl3.UUID, tables[0].UUID)
	a.Equal(p2.Email, tables[1].Email)
	a.Equal(tbl2.UUID, tables[1].UUID)

	// sanity check
	a.NotEqual(p2.Email, p3.Email)
}

func TestPlayer_CloneTable(t *testing.T) {
	a := assert.New(t)

	admin, source := playerAndTable()
	admin.IsSiteAdmin = true // bypass create-table cooldown
	a.NoError(admin.Save(cbg))

	adminPT, _ := admin.GetPlayerTable(cbg, source)
	adminPT.TableStake = 5000
	a.NoError(adminPT.Save(cbg))
	a.NoError(adminPT.AdjustBalance(cbg, 100, "won pot", nil))

	p2 := player()
	pt2, _ := p2.Join(cbg, source)
	pt2.TableStake = 3000
	pt2.IsBlocked = true
	a.NoError(pt2.Save(cbg))
	a.NoError(pt2.AdjustBalance(cbg, -50, "lost pot", nil))

	p3 := player()
	pt3, _ := p3.Join(cbg, source)
	pt3.CanStart = true
	pt3.CanRestart = true
	pt3.CanTerminate = true
	pt3.IsTableAdmin = true
	a.NoError(pt3.Save(cbg))

	p4 := player()
	_, _ = p4.Join(cbg, source)

	cloned, err := admin.CloneTable(cbg, source, "Cloned Table")
	a.NoError(err)
	a.NotNil(cloned)
	a.Equal("Cloned Table", cloned.Name)
	a.NotEqual(source.UUID, cloned.UUID)
	a.Equal(admin.ID, cloned.PlayerID)

	players, err := cloned.GetPlayers(cbg)
	a.NoError(err)
	a.Equal(4, len(players))

	byID := make(map[int64]*PlayerTable, len(players))
	for _, pt := range players {
		byID[pt.PlayerID] = pt
		a.Equal(0, pt.Balance, "player %d balance must be zero", pt.PlayerID)
		a.False(pt.Active, "player %d must start in sit-out state", pt.PlayerID)
	}

	a.True(byID[admin.ID].IsTableAdmin)
	a.Equal(5000, byID[admin.ID].TableStake)
	a.False(byID[admin.ID].IsBlocked)

	a.Equal(3000, byID[p2.ID].TableStake)
	a.True(byID[p2.ID].IsBlocked)
	a.False(byID[p2.ID].IsTableAdmin)

	a.True(byID[p3.ID].IsTableAdmin)
	a.True(byID[p3.ID].CanStart)
	a.True(byID[p3.ID].CanRestart)
	a.True(byID[p3.ID].CanTerminate)

	a.Equal(2000, byID[p4.ID].TableStake)
	a.False(byID[p4.ID].IsTableAdmin)

	// source table is untouched
	srcPlayers, _ := source.GetPlayers(cbg)
	a.Equal(4, len(srcPlayers))
	for _, sp := range srcPlayers {
		switch sp.PlayerID {
		case admin.ID:
			a.Equal(100, sp.Balance)
			a.True(sp.Active)
		case p2.ID:
			a.Equal(-50, sp.Balance)
		}
	}
}

func TestPlayer_CloneTable_randomizesOrder(t *testing.T) {
	a := assert.New(t)

	admin, source := playerAndTable()
	admin.IsSiteAdmin = true
	a.NoError(admin.Save(cbg))

	for i := 0; i < 7; i++ {
		_, _ = player().Join(cbg, source)
	}

	srcPlayers, _ := source.GetPlayers(cbg)
	srcOrder := make([]int64, len(srcPlayers))
	for i, sp := range srcPlayers {
		srcOrder[i] = sp.PlayerID
	}

	// Across several clones, ordering should differ from the source at least once.
	differs := false
	for i := 0; i < 8 && !differs; i++ {
		cloned, err := admin.CloneTable(cbg, source, fmt.Sprintf("clone-%d", i))
		a.NoError(err)

		clonedPlayers, _ := cloned.GetPlayers(cbg)
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

func TestPlayer_CloneTable_notTableAdmin(t *testing.T) {
	_, source := playerAndTable()

	p2 := player()
	_, _ = p2.Join(cbg, source)

	_, err := p2.CloneTable(cbg, source, "Clone")
	var ue UserError
	assert.True(t, errors.As(err, &ue))
	assert.Equal(t, "only a table admin can clone a table", err.Error())
}

func TestPlayer_CloneTable_playerNotAtTable(t *testing.T) {
	_, source := playerAndTable()
	stranger := player()

	_, err := stranger.CloneTable(cbg, source, "Clone")
	var ue UserError
	assert.True(t, errors.As(err, &ue))
	assert.Equal(t, "only a table admin can clone a table", err.Error())
}

func TestPlayer_CloneTable_siteAdminNotAtTable(t *testing.T) {
	a := assert.New(t)

	_, source := playerAndTable()

	siteAdmin := player()
	siteAdmin.IsSiteAdmin = true
	a.NoError(siteAdmin.Save(cbg))

	cloned, err := siteAdmin.CloneTable(cbg, source, "Clone")
	a.NoError(err)
	a.NotNil(cloned)
	a.Equal("Clone", cloned.Name)
	a.Equal(siteAdmin.ID, cloned.PlayerID)

	srcPlayers, _ := source.GetPlayers(cbg)
	clonedPlayers, _ := cloned.GetPlayers(cbg)
	a.Equal(len(srcPlayers), len(clonedPlayers))
}

func TestTable_Save(t *testing.T) {
	a := assert.New(t)

	_, table := playerAndTable()
	a.False(table.Deleted)

	modifiedBefore := table.Modified

	origName := table.Name
	table.Name = origName + "-updated"
	table.Deleted = true
	a.NoError(table.Save(cbg))

	table, err := GetTableByUUID(cbg, table.UUID)
	a.NoError(err)
	a.NotEqual(origName, table.Name)
	a.True(table.Deleted)
	a.False(table.Modified.Before(modifiedBefore), "modified should not be before the original modified time")
}
