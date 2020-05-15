package table

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
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

	_ = pt4.SetActive(cbg, false)

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