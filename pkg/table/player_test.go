package table

import (
	"context"
	"database/sql"
	"fmt"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/db"
	"strconv"
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
	assert.Equal(t, ErrAccountNotVerified, err)
	assert.Nil(t, player2)

	// verify the account
	{
		p2, _ := GetPlayerByEmail(cbg, email)
		p2.Status = PlayerStatusVerified
		assert.NoError(t, p2.Save(cbg))
	}

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
	assert.Equal(t, table.PlayerID, player.ID)

	table2, err := player.CreateTable(cbg, "my table")
	assert.EqualError(t, err, "you must wait before you create another table")
	assert.Nil(t, table2)

	const query = `
UPDATE tables
SET created = (now() at time zone 'utc') - interval '61 second'
WHERE uuid = $1`
	_, err = db.Instance().Exec(query, table.UUID)
	assert.NoError(t, err)

	table2, err = player.CreateTable(cbg, "my table")
	assert.NoError(t, err)
	assert.NotNil(t, table2)
	assert.NotEqual(t, table2.UUID, table.UUID)
	assert.Equal(t, table2.PlayerID, player.ID)

	table3, err := player.CreateTable(cbg, "my table")
	assert.EqualError(t, err, "you must wait before you create another table")
	assert.Nil(t, table3)
	player.IsSiteAdmin = true
	table3, err = player.CreateTable(cbg, "my table")
	assert.NoError(t, err)
	assert.NotNil(t, table3)

	table, err = GetTableByUUID(cbg, table.UUID)
	assert.NoError(t, err)
	assert.Equal(t, "my table", table.Name)
	assert.Equal(t, player.ID, table.PlayerID)
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
	p.IsSiteAdmin = true // to rapidly create tables
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

func verifiedPlayer() *Player {
	p := player()

	p.Status = PlayerStatusVerified
	_ = p.Save(cbg)

	return p
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
	p := player()
	_ = player()
	_ = player()

	players, err := GetPlayers(cbg, 0, 4)
	assert.NoError(t, err)
	assert.Equal(t, len(players), 4)

	players, err = GetPlayersWithSearch(cbg, "", 0, 4)
	assert.NoError(t, err)
	assert.Equal(t, len(players), 4)

	players, err = GetPlayers(cbg, 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, len(players), 1)

	players, err = GetPlayersWithSearch(cbg, strconv.FormatInt(p.ID, 10), 0, 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(players))

	players, err = GetPlayersWithSearch(cbg, "test-", 0, 4)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(players))

	players, err = GetPlayersWithSearch(cbg, p.Email, 0, 4)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(players))
}

func TestPlayer_SetPassword(t *testing.T) {
	const newPassword = "my-new-password"
	a := assert.New(t)
	p := verifiedPlayer()
	player, err := GetPlayerByEmailAndPassword(context.Background(), p.Email, newPassword)
	a.Nil(player)
	a.EqualError(err, "invalid email address and/or password")

	a.NoError(p.SetPassword(newPassword))

	// still doesn't work because we didn't call save
	player, err = GetPlayerByEmailAndPassword(context.Background(), p.Email, newPassword)
	a.Nil(player)
	a.EqualError(err, "invalid email address and/or password")

	a.NoError(p.Save(context.Background()))

	// now the new password works
	player, err = GetPlayerByEmailAndPassword(context.Background(), p.Email, newPassword)
	a.NotNil(player)
	a.NoError(err)
}

func TestPlayer_ResetPassword(t *testing.T) {
	a := assert.New(t)

	p := verifiedPlayer()
	differentPlayer := verifiedPlayer()

	tkn, err := p.CreatePasswordResetRequest(cbg)
	a.NoError(err)
	a.Equal(20, len(tkn))

	// test a bad token
	a.EqualError(p.ResetPassword(cbg, "test", "bad-token"), "could not reset the password")

	// ensure token only works for the correct player
	a.EqualError(differentPlayer.ResetPassword(cbg, "test", tkn), "could not reset the password")

	// verify it works
	a.NoError(p.ResetPassword(cbg, "my new password", tkn))

	p2, err := GetPlayerByEmailAndPassword(cbg, p.Email, "my new password")
	a.NoError(err)
	a.NotNil(p2)

	// ensure token can only be used once
	a.EqualError(p.ResetPassword(cbg, "another new password", tkn), "could not reset the password")
}

// ensure that a reset password request is only valid for one hour
func TestPlayer_ResetPassword_expired(t *testing.T) {
	a := assert.New(t)

	p := player()
	token, err := p.CreatePasswordResetRequest(cbg)
	a.NoError(err)

	a.NoError(IsPasswordResetTokenValid(cbg, token))

	const query = `
UPDATE player_tokens
SET created = (NOW() AT TIME ZONE 'UTC') - INTERVAL '2 hour'
WHERE token = $1
`

	_, err = db.Instance().Exec(query, token)
	a.NoError(err)

	a.Equal(ErrTokenExpired, IsPasswordResetTokenValid(cbg, token))

	a.EqualError(p.ResetPassword(cbg, "my new password", token), "could not reset the password")
}

func TestPlayer_accountVerification(t *testing.T) {
	a := assert.New(t)

	p := player()
	a.NoError(p.SetPassword("test"))
	a.NoError(p.Save(context.Background()))
	a.NotEqual(PlayerStatusVerified, p.Status)

	_, err := GetPlayerByEmailAndPassword(cbg, p.Email, "test")
	a.Equal(ErrAccountNotVerified, err)

	token, err := p.CreateAccountVerificationToken(cbg)
	a.NoError(err)

	a.EqualError(VerifyAccount(cbg, "bad-token"), "token is expired")
	a.NoError(VerifyAccount(cbg, token))

	p2, err := GetPlayerByEmailAndPassword(cbg, p.Email, "test")
	a.NoError(err)
	a.NotNil(p2)

	// can't re-use token
	a.EqualError(VerifyAccount(cbg, token), "token is expired")
}

func TestPlayer_Delete(t *testing.T) {
	p := player()
	email := p.Email
	displayName := p.DisplayName

	a := assert.New(t)
	a.NoError(p.Delete(cbg))
	a.NotEqual(email, p.Email)
	a.NotEqual(displayName, p.DisplayName)

	oldRecord, _ := GetPlayerByEmail(cbg, email)
	a.Nil(oldRecord)

	newRecord, _ := GetPlayerByEmail(cbg, p.Email)
	a.NotEqual(email, newRecord.Email)
	a.NotEqual(displayName, newRecord.DisplayName)
}
