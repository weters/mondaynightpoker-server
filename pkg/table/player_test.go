package table

import (
	"context"
	"database/sql"
	"fmt"
	"mondaynightpoker-server/internal/util"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var cbg = context.Background()

func TestCreatePlayer(t *testing.T) {
	remoteAddr := fmt.Sprintf("127.0.0.1:%d", time.Now().UnixNano())

	at, err := LastPlayerCreatedAt(cbg, remoteAddr)
	assert.NoError(t, err)
	assert.True(t, at.IsZero())

	before := time.Now()

	email := util.RandomEmail()
	player, err := CreatePlayer(cbg, email, "test-player", "password", remoteAddr)
	assert.NoError(t, err)
	assert.NotNil(t, player)
	assert.Greater(t, player.ID, int64(0))

	at, err = LastPlayerCreatedAt(cbg, remoteAddr)
	assert.NoError(t, err)
	assert.True(t, at.After(before))

	at, err = LastPlayerCreatedAt(cbg, "::1")
	assert.NoError(t, err)
	assert.True(t, at.IsZero())

	player2, err := CreatePlayer(cbg, email, "test-player", "password", remoteAddr)
	assert.Equal(t, err, ErrDuplicateKey)
	assert.Nil(t, player2)

	player2, err = CreatePlayer(cbg, util.RandomEmail(), "test-player", "password2", remoteAddr)
	assert.NoError(t, err)
	assert.NotNil(t, player2)
	assert.Greater(t, player2.ID, player.ID)

	player2, err = GetPlayerByEmailAndPassword(cbg, email, "bad-password")
	assert.Equal(t, ErrInvalidEmailOrPassword, err)
	assert.Nil(t, player2)

	player2, err = GetPlayerByEmailAndPassword(cbg, email+"-not-found", "password")
	assert.Equal(t, ErrInvalidEmailOrPassword, err)
	assert.Nil(t, player2)

	player2, err = GetPlayerByEmailAndPassword(cbg, email, "password")
	assert.NoError(t, err)
	assert.NotNil(t, player2)

	// test case-insensitive email
	player2, err = GetPlayerByEmailAndPassword(cbg, strings.ToUpper(email), "password")
	assert.NoError(t, err)
	assert.NotNil(t, player2)

	// ensure you can't create a duplicate player with a case-insensitive email
	_, err = CreatePlayer(cbg, strings.ToUpper(email), "Display", "password", "[::1]")
	assert.Equal(t, ErrDuplicateKey, err)
}

func TestPlayerByID(t *testing.T) {
	p := player()
	player, err := GetPlayerByID(cbg, p.ID)
	assert.NoError(t, err)
	assert.Equal(t, p.ID, player.ID)

	player, err = GetPlayerByID(cbg, 0)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, player)
}

func TestPlayer_CreateTable(t *testing.T) {
	player := player()
	table, err := player.CreateTable(cbg, "my table")
	assert.NoError(t, err)
	assert.NotNil(t, table)
	assert.NotEmpty(t, table.UUID)

	table2, err := player.CreateTable(cbg, "my table")
	assert.NoError(t, err)
	assert.NotNil(t, table2)
	assert.NotEqual(t, table2.UUID, table.UUID)
}

func TestPlayer_Join(t *testing.T) {
	p1 := player()
	table, _ := p1.CreateTable(cbg, "my table")

	before := time.Now()
	p2 := player()
	playerTable, err := p2.Join(cbg, table)
	assert.NoError(t, err)
	assert.NotNil(t, playerTable)
	assert.Greater(t, playerTable.ID, int64(0))
	assert.True(t, playerTable.Created.After(before))
	assert.True(t, playerTable.Updated.After(before))

	playerTable, err = p2.Join(cbg, table)
	assert.Equal(t, ErrDuplicateKey, err)
	assert.Nil(t, playerTable)
}

func TestPlayer_SetIsSiteAdmin(t *testing.T) {
	p := player()
	assert.False(t, p.IsSiteAdmin)
	assert.Equal(t, p.Created, p.Updated)
	assert.NoError(t, p.SetIsSiteAdmin(cbg, true))
	assert.True(t, p.IsSiteAdmin)
	assert.True(t, p.Updated.After(p.Created))

	p1, _ := GetPlayerByID(cbg, p.ID)
	assert.True(t, p1.IsSiteAdmin)
}

func TestPlayer_GetTables(t *testing.T) {
	p := player()
	tbl1, _ := p.CreateTable(cbg, "Table 1")
	tbl2, _ := p.CreateTable(cbg, "Table 2")
	tbl3, _ := p.CreateTable(cbg, "Table 3")

	pt, _ := p.GetPlayerTable(cbg, tbl1)
	_ = pt.AdjustBalance(cbg, 25, "", nil)

	p2 := player()
	_, _ = p2.Join(cbg, tbl2)
	_, _ = p2.Join(cbg, tbl1)

	tables, err := p.GetTables(cbg, 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(tables))
	assert.Equal(t, tbl2.UUID, tables[0].UUID)

	tables, err = p.GetTables(cbg, 0, 99)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(tables))
	assert.Equal(t, tbl3.UUID, tables[0].UUID)
	assert.Equal(t, 0, tables[0].Balance)
	assert.Equal(t, tbl2.UUID, tables[1].UUID)
	assert.Equal(t, 0, tables[1].Balance)
	assert.Equal(t, tbl1.UUID, tables[2].UUID)
	assert.Equal(t, 25, tables[2].Balance)

	tables, err = p2.GetTables(cbg, 0, 99)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(tables))
	assert.Equal(t, tbl1.UUID, tables[0].UUID)
	assert.Equal(t, tbl2.UUID, tables[1].UUID)
}

func player() *Player {
	player, err := CreatePlayer(cbg, util.RandomEmail(), "test-player", "", "127.0.0.1")
	if err != nil {
		panic(err)
	}

	return player
}

func TestPlayer_Save(t *testing.T) {
	newEmail := util.RandomEmail()

	p := player()
	p.Email = newEmail
	p.IsSiteAdmin = true
	p.DisplayName = "New Display Name"

	assert.NoError(t, p.Save(cbg))

	p1, _ := GetPlayerByID(cbg, p.ID)
	assert.Equal(t, newEmail, p1.Email)
	assert.Equal(t, true, p.IsSiteAdmin)
	assert.Equal(t, "New Display Name", p.DisplayName)
	assert.True(t, p1.Updated.After(p1.Created))
}

func TestGetPlayers(t *testing.T) {
	_ = player()
	_ = player()
	_ = player()
	_ = player()

	players, err := GetPlayers(cbg, 0, 4)
	assert.NoError(t, err)
	assert.Equal(t, len(players), 4)

	players, err = GetPlayers(cbg, 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, len(players), 1)
}
