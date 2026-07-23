package model

import (
	"context"
	"database/sql"
	"fmt"
	"mondaynightpoker-server/internal/util"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePlayer(t *testing.T) {
	remoteAddr := fmt.Sprintf("127.0.0.1:%d", time.Now().UnixNano())

	at, err := testRepos.Players.LastPlayerCreatedAt(cbg, remoteAddr)
	assert.NoError(t, err)
	assert.True(t, at.IsZero())

	before := time.Now()

	email := util.RandomEmail()
	player, err := testRepos.Players.CreatePlayer(cbg, email, "test-player", "password", remoteAddr)
	assert.NoError(t, err)
	assert.NotNil(t, player)
	assert.Greater(t, player.ID, int64(0))

	at, err = testRepos.Players.LastPlayerCreatedAt(cbg, remoteAddr)
	assert.NoError(t, err)
	assert.True(t, at.After(before))

	at, err = testRepos.Players.LastPlayerCreatedAt(cbg, "::1")
	assert.NoError(t, err)
	assert.True(t, at.IsZero())

	player2, err := testRepos.Players.CreatePlayer(cbg, email, "test-player", "password", remoteAddr)
	assert.Equal(t, err, ErrDuplicateKey)
	assert.Nil(t, player2)

	player2, err = testRepos.Players.CreatePlayer(cbg, util.RandomEmail(), "test-player", "password2", remoteAddr)
	assert.NoError(t, err)
	assert.NotNil(t, player2)
	assert.Greater(t, player2.ID, player.ID)

	player2, err = testRepos.Players.GetPlayerByEmailAndPassword(cbg, email, "bad-password")
	assert.Equal(t, ErrInvalidEmailOrPassword, err)
	assert.Nil(t, player2)

	player2, err = testRepos.Players.GetPlayerByEmailAndPassword(cbg, email+"-not-found", "password")
	assert.Equal(t, ErrInvalidEmailOrPassword, err)
	assert.Nil(t, player2)

	player2, err = testRepos.Players.GetPlayerByEmailAndPassword(cbg, email, "password")
	assert.Equal(t, ErrAccountNotVerified, err)
	assert.Nil(t, player2)

	// verify the account
	{
		p2, _ := testRepos.Players.GetPlayerByEmail(cbg, email)
		p2.Status = PlayerStatusVerified
		assert.NoError(t, testRepos.Players.Save(cbg, p2))
	}

	player2, err = testRepos.Players.GetPlayerByEmailAndPassword(cbg, email, "password")
	assert.NoError(t, err)
	assert.NotNil(t, player2)

	// test case-insensitive email
	player2, err = testRepos.Players.GetPlayerByEmailAndPassword(cbg, strings.ToUpper(email), "password")
	assert.NoError(t, err)
	assert.NotNil(t, player2)

	// ensure you can't create a duplicate player with a case-insensitive email
	_, err = testRepos.Players.CreatePlayer(cbg, strings.ToUpper(email), "Display", "password", "[::1]")
	assert.Equal(t, ErrDuplicateKey, err)
}

func TestPlayerByID(t *testing.T) {
	p := player()
	player, err := testRepos.Players.GetPlayerByID(cbg, p.ID)
	assert.NoError(t, err)
	assert.Equal(t, p.ID, player.ID)

	player, err = testRepos.Players.GetPlayerByID(cbg, 0)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, player)
}

func TestPlayer_CreateTable(t *testing.T) {
	player := player()
	table, err := testRepos.Tables.CreateTable(cbg, player, "my table")
	assert.NoError(t, err)
	assert.NotNil(t, table)
	assert.NotEmpty(t, table.UUID)
	assert.Equal(t, table.PlayerID, player.ID)

	table2, err := testRepos.Tables.CreateTable(cbg, player, "my table")
	assert.EqualError(t, err, "you must wait before you create another table")
	assert.Nil(t, table2)

	const query = `
UPDATE tables
SET created = (now() at time zone 'utc') - interval '61 second'
WHERE uuid = $1`
	_, err = testDB.Exec(query, table.UUID)
	assert.NoError(t, err)

	table2, err = testRepos.Tables.CreateTable(cbg, player, "my table")
	assert.NoError(t, err)
	assert.NotNil(t, table2)
	assert.NotEqual(t, table2.UUID, table.UUID)
	assert.Equal(t, table2.PlayerID, player.ID)

	table3, err := testRepos.Tables.CreateTable(cbg, player, "my table")
	assert.EqualError(t, err, "you must wait before you create another table")
	assert.Nil(t, table3)
	player.IsSiteAdmin = true
	table3, err = testRepos.Tables.CreateTable(cbg, player, "my table")
	assert.NoError(t, err)
	assert.NotNil(t, table3)

	table, err = testRepos.Tables.GetTableByUUID(cbg, table.UUID)
	assert.NoError(t, err)
	assert.Equal(t, "my table", table.Name)
	assert.Equal(t, player.ID, table.PlayerID)
}

func TestPlayer_Join(t *testing.T) {
	p1 := player()
	table, _ := testRepos.Tables.CreateTable(cbg, p1, "my table")

	before := time.Now()
	p2 := player()
	playerTable, err := testRepos.Tables.Join(cbg, p2, table)
	assert.NoError(t, err)
	assert.NotNil(t, playerTable)
	assert.Greater(t, playerTable.ID, int64(0))
	assert.True(t, playerTable.Created.After(before))
	assert.True(t, playerTable.Updated.After(before))

	playerTable, err = testRepos.Tables.Join(cbg, p2, table)
	assert.Equal(t, ErrDuplicateKey, err)
	assert.Nil(t, playerTable)
}

func TestPlayer_SetIsSiteAdmin(t *testing.T) {
	p := player()
	assert.False(t, p.IsSiteAdmin)
	assert.Equal(t, p.Created, p.Updated)
	assert.NoError(t, testRepos.Players.SetIsSiteAdmin(cbg, p, true))
	assert.True(t, p.IsSiteAdmin)
	assert.True(t, p.Updated.After(p.Created))

	p1, _ := testRepos.Players.GetPlayerByID(cbg, p.ID)
	assert.True(t, p1.IsSiteAdmin)
}

func verifiedPlayer() *Player {
	p := player()

	p.Status = PlayerStatusVerified
	_ = testRepos.Players.Save(cbg, p)

	return p
}

func player() *Player {
	player, err := testRepos.Players.CreatePlayer(cbg, util.RandomEmail(), "test-player", "", "127.0.0.1")
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

	assert.NoError(t, testRepos.Players.Save(cbg, p))

	p1, _ := testRepos.Players.GetPlayerByID(cbg, p.ID)
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

	players, err := testRepos.Players.GetPlayers(cbg, 0, 4)
	assert.NoError(t, err)
	assert.Equal(t, len(players), 4)

	players, err = testRepos.Players.GetPlayersWithSearch(cbg, "", 0, 4)
	assert.NoError(t, err)
	assert.Equal(t, len(players), 4)

	players, err = testRepos.Players.GetPlayers(cbg, 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, len(players), 1)

	players, err = testRepos.Players.GetPlayersWithSearch(cbg, strconv.FormatInt(p.ID, 10), 0, 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(players))

	players, err = testRepos.Players.GetPlayersWithSearch(cbg, "test-", 0, 4)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(players))

	players, err = testRepos.Players.GetPlayersWithSearch(cbg, p.Email, 0, 4)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(players))
}

func TestPlayer_SetPassword(t *testing.T) {
	const newPassword = "my-new-password"
	a := assert.New(t)
	p := verifiedPlayer()
	player, err := testRepos.Players.GetPlayerByEmailAndPassword(context.Background(), p.Email, newPassword)
	a.Nil(player)
	a.EqualError(err, "invalid email address and/or password")

	a.NoError(p.SetPassword(newPassword))

	// still doesn't work because we didn't call save
	player, err = testRepos.Players.GetPlayerByEmailAndPassword(context.Background(), p.Email, newPassword)
	a.Nil(player)
	a.EqualError(err, "invalid email address and/or password")

	a.NoError(testRepos.Players.Save(context.Background(), p))

	// now the new password works
	player, err = testRepos.Players.GetPlayerByEmailAndPassword(context.Background(), p.Email, newPassword)
	a.NotNil(player)
	a.NoError(err)
}

func TestPlayer_ResetPassword(t *testing.T) {
	a := assert.New(t)

	p := verifiedPlayer()
	differentPlayer := verifiedPlayer()

	tkn, err := testRepos.Players.CreatePasswordResetRequest(cbg, p)
	a.NoError(err)
	a.Equal(20, len(tkn))

	// test a bad token
	a.EqualError(testRepos.Players.ResetPassword(cbg, p, "test", "bad-token"), "could not reset the password")

	// ensure token only works for the correct player
	a.EqualError(testRepos.Players.ResetPassword(cbg, differentPlayer, "test", tkn), "could not reset the password")

	// verify it works
	a.NoError(testRepos.Players.ResetPassword(cbg, p, "my new password", tkn))

	p2, err := testRepos.Players.GetPlayerByEmailAndPassword(cbg, p.Email, "my new password")
	a.NoError(err)
	a.NotNil(p2)

	// ensure token can only be used once
	a.EqualError(testRepos.Players.ResetPassword(cbg, p, "another new password", tkn), "could not reset the password")
}

// ensure that a reset password request is only valid for one hour
func TestPlayer_ResetPassword_expired(t *testing.T) {
	a := assert.New(t)

	p := player()
	token, err := testRepos.Players.CreatePasswordResetRequest(cbg, p)
	a.NoError(err)

	a.NoError(testRepos.Players.IsPasswordResetTokenValid(cbg, token))

	const query = `
UPDATE player_tokens
SET created = (NOW() AT TIME ZONE 'UTC') - INTERVAL '2 hour'
WHERE token = $1
`

	_, err = testDB.Exec(query, token)
	a.NoError(err)

	a.Equal(ErrTokenExpired, testRepos.Players.IsPasswordResetTokenValid(cbg, token))

	a.EqualError(testRepos.Players.ResetPassword(cbg, p, "my new password", token), "could not reset the password")
}

func TestPlayer_accountVerification(t *testing.T) {
	a := assert.New(t)

	p := player()
	a.NoError(p.SetPassword("test"))
	a.NoError(testRepos.Players.Save(context.Background(), p))
	a.NotEqual(PlayerStatusVerified, p.Status)

	_, err := testRepos.Players.GetPlayerByEmailAndPassword(cbg, p.Email, "test")
	a.Equal(ErrAccountNotVerified, err)

	token, err := testRepos.Players.CreateAccountVerificationToken(cbg, p)
	a.NoError(err)

	a.EqualError(testRepos.Players.VerifyAccount(cbg, "bad-token"), "token is expired")
	a.NoError(testRepos.Players.VerifyAccount(cbg, token))

	p2, err := testRepos.Players.GetPlayerByEmailAndPassword(cbg, p.Email, "test")
	a.NoError(err)
	a.NotNil(p2)

	// can't re-use token
	a.EqualError(testRepos.Players.VerifyAccount(cbg, token), "token is expired")
}

func TestGameTypeGroup(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Bourré", "Bourre"},
		{"Bourré (Five Suit)", "Bourre"},
		{"4-Card Little L (trade: 0, 2)", "Little L"},
		{"5-Card Little L (trade: 0, 1, 3)", "Little L"},
		{"Acey Deucey", "Acey Deucey"},
		{"Acey Deucey (Continuous Shoe)", "Acey Deucey"},
		{"Texas Hold'em (${25}/${50})", "Texas Hold'em"},
		{"Limit Texas Hold'em (${50}/${100})", "Texas Hold'em"},
		{"Pineapple (${25}/${50})", "Texas Hold'em"},
		{"Lazy Pineapple (${25}/${50})", "Texas Hold'em"},
		{"Pass the Poop, Standard Edition", "Pass the Poop"},
		{"Pass the Poop, Diarrhea Edition (with Blocks)", "Pass the Poop"},
		{"2-Card Guts", "Guts"},
		{"Bloody 3-Card Guts with Trades", "Guts"},
		{"Seven-Card Stud", "Seven Card"},
		{"Baseball", "Seven Card"},
		{"Follow the Queen", "Seven Card"},
		{"High Chicago", "Seven Card"},
		{"Low Card Wild", "Seven Card"},
		{"7 Card Chiggs", "Seven Card"},
		{"7-Card TJ", "Seven Card"},
		{"Coupons and Clippings", "Seven Card"},
		{"Unknown Game", "Unknown Game"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, GameTypeGroup(tt.input))
		})
	}
}

func TestGetPlayerStats(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true // to rapidly create tables
	tbl1, _ := testRepos.Tables.CreateTable(cbg, p, "Stats Table 1")
	tbl2, _ := testRepos.Tables.CreateTable(cbg, p, "Stats Table 2")

	// Create games and end them with balance adjustments
	// Use display names as they would be stored in the DB
	game1, _ := testRepos.Games.CreateGame(cbg, tbl1, "Bourré")
	_ = testRepos.Games.EndGame(cbg, game1, nil, map[int64]int{p.ID: 100})

	game2, _ := testRepos.Games.CreateGame(cbg, tbl1, "Texas Hold'em (${25}/${50})")
	_ = testRepos.Games.EndGame(cbg, game2, nil, map[int64]int{p.ID: -50})

	game3, _ := testRepos.Games.CreateGame(cbg, tbl2, "Bourré (Five Suit)")
	_ = testRepos.Games.EndGame(cbg, game3, nil, map[int64]int{p.ID: 200})

	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	stats, err := testRepos.Players.GetPlayerStats(cbg, p.ID, from, to)
	a.NoError(err)
	a.Equal(2, stats.TablesJoined)
	a.Equal(3, stats.GamesPlayed)
	a.Equal(250, stats.TotalWinnings)
	// Both Bourré variants should be grouped under "Bourre"
	a.Equal(300, stats.WinningsByGame["Bourre"])
	a.Equal(-50, stats.WinningsByGame["Texas Hold'em"])
	// Game counts by type
	a.Equal(2, stats.GamesCountByType["Bourre"])
	a.Equal(1, stats.GamesCountByType["Texas Hold'em"])

	// Test with narrow date range that excludes everything
	future := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	futureEnd := time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC)
	stats, err = testRepos.Players.GetPlayerStats(cbg, p.ID, future, futureEnd)
	a.NoError(err)
	a.Equal(0, stats.TablesJoined)
	a.Equal(0, stats.GamesPlayed)
	a.Equal(0, stats.TotalWinnings)
	a.Equal(0, len(stats.WinningsByGame))
	a.Equal(0, len(stats.GamesCountByType))
}

// TestGetPlayerStats_ExcludesDeletedTables pins the invariant that a soft-deleted
// table contributes nothing to any player stat. The scalar aggregates are the easy
// ones to miss: they return no table record, so a deleted table leaks through them
// as a number rather than as a visible row.
func TestGetPlayerStats_ExcludesDeletedTables(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true // to rapidly create tables
	live, _ := testRepos.Tables.CreateTable(cbg, p, "Stats Live Table")
	gone, _ := testRepos.Tables.CreateTable(cbg, p, "Stats Deleted Table")

	liveGame, _ := testRepos.Games.CreateGame(cbg, live, "Bourré")
	_ = testRepos.Games.EndGame(cbg, liveGame, nil, map[int64]int{p.ID: 100})

	goneGame, _ := testRepos.Games.CreateGame(cbg, gone, "Texas Hold'em (${25}/${50})")
	_ = testRepos.Games.EndGame(cbg, goneGame, nil, map[int64]int{p.ID: 700})

	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	// while both tables are live, both contribute
	stats, err := testRepos.Players.GetPlayerStats(cbg, p.ID, from, to)
	a.NoError(err)
	a.Equal(2, stats.TablesJoined)
	a.Equal(2, stats.GamesPlayed)
	a.Equal(800, stats.TotalWinnings)

	gone.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, gone))

	// once soft-deleted, the table is absent from every figure
	stats, err = testRepos.Players.GetPlayerStats(cbg, p.ID, from, to)
	a.NoError(err)
	a.Equal(1, stats.TablesJoined)
	a.Equal(1, stats.GamesPlayed)
	a.Equal(100, stats.TotalWinnings)
	a.Equal(100, stats.WinningsByGame["Bourre"])
	a.Equal(1, stats.GamesCountByType["Bourre"])
	a.NotContains(stats.WinningsByGame, "Texas Hold'em")
	a.NotContains(stats.GamesCountByType, "Texas Hold'em")
}

// TestGetPlayerProfile_DeletedTableIsAbsentEverywhere checks the three parts of a
// profile agree with each other. Before the fix, stats summed deleted-table money
// while tables and graphData excluded it, so totalWinnings named a figure that no
// visible session accounted for.
func TestGetPlayerProfile_DeletedTableIsAbsentEverywhere(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true // to rapidly create tables
	live, _ := testRepos.Tables.CreateTable(cbg, p, "Profile Live Table")
	gone, _ := testRepos.Tables.CreateTable(cbg, p, "Profile Deleted Table")

	liveGame, _ := testRepos.Games.CreateGame(cbg, live, "bourre")
	_ = testRepos.Games.EndGame(cbg, liveGame, nil, map[int64]int{p.ID: 250})

	goneGame, _ := testRepos.Games.CreateGame(cbg, gone, "bourre")
	_ = testRepos.Games.EndGame(cbg, goneGame, nil, map[int64]int{p.ID: 900})

	gone.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, gone))

	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	profile, err := testRepos.Players.GetPlayerProfile(cbg, p.ID, from, to, 0, 100)
	a.NoError(err)

	a.Equal(250, profile.Stats.TotalWinnings)
	a.Len(profile.Tables, 1)
	a.Equal(live.UUID, profile.Tables[0].UUID)
	a.Len(profile.GraphData, 1)

	// the parts reconcile: the sessions on show account for the reported total
	var graphTotal int
	for _, gp := range profile.GraphData {
		graphTotal += gp.Balance
	}
	a.Equal(profile.Stats.TotalWinnings, graphTotal)
}

func TestGetPlayerTablesFiltered(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true
	tbl1, _ := testRepos.Tables.CreateTable(cbg, p, "Filtered Table 1")
	tbl2, _ := testRepos.Tables.CreateTable(cbg, p, "Filtered Table 2")
	tbl3, _ := testRepos.Tables.CreateTable(cbg, p, "Filtered Table 3")

	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	tables, err := testRepos.Players.GetPlayerTablesFiltered(cbg, p.ID, from, to, 0, 100)
	a.NoError(err)
	a.Equal(3, len(tables))
	a.Equal(tbl3.UUID, tables[0].UUID)
	a.Equal(tbl2.UUID, tables[1].UUID)
	a.Equal(tbl1.UUID, tables[2].UUID)

	// Test pagination
	tables, err = testRepos.Players.GetPlayerTablesFiltered(cbg, p.ID, from, to, 0, 2)
	a.NoError(err)
	a.Equal(2, len(tables))

	tables, err = testRepos.Players.GetPlayerTablesFiltered(cbg, p.ID, from, to, 2, 2)
	a.NoError(err)
	a.Equal(1, len(tables))

	// Test narrow date range
	future := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	futureEnd := time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC)
	tables, err = testRepos.Players.GetPlayerTablesFiltered(cbg, p.ID, future, futureEnd, 0, 100)
	a.NoError(err)
	a.Equal(0, len(tables))
}

func TestGetLeaderboard(t *testing.T) {
	a := assert.New(t)

	caller := player()
	caller.IsSiteAdmin = true // to rapidly create tables

	// caller belongs to tableA and tableB (as their creator)
	tableA, _ := testRepos.Tables.CreateTable(cbg, caller, "LB Table A")
	tableB, _ := testRepos.Tables.CreateTable(cbg, caller, "LB Table B")

	other := player()
	_, err := testRepos.Tables.Join(cbg, other, tableA)
	require.NoError(t, err)

	// a table the caller is NOT a member of, run by a third player
	outsider := player()
	outsider.IsSiteAdmin = true
	tableD, _ := testRepos.Tables.CreateTable(cbg, outsider, "LB Table D")
	_, err = testRepos.Tables.Join(cbg, other, tableD)
	require.NoError(t, err)

	// games
	gA, _ := testRepos.Games.CreateGame(cbg, tableA, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, gA, nil, map[int64]int{caller.ID: 100, other.ID: -100}))
	gB, _ := testRepos.Games.CreateGame(cbg, tableB, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, gB, nil, map[int64]int{caller.ID: 50}))
	gD, _ := testRepos.Games.CreateGame(cbg, tableD, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, gD, nil, map[int64]int{outsider.ID: 300, other.ID: 20}))

	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	entries, err := testRepos.Players.GetLeaderboard(cbg, caller.ID, from, to)
	a.NoError(err)

	byPlayer := make(map[int64]*LeaderboardEntry)
	for _, e := range entries {
		byPlayer[e.PlayerID] = e
	}

	// the outsider is only in tableD (out of scope) and must not appear
	a.NotContains(byPlayer, outsider.ID)

	// caller: net 150 across two tables, two games
	if a.Contains(byPlayer, caller.ID) {
		a.Equal(150, byPlayer[caller.ID].NetWinnings)
		a.Equal(2, byPlayer[caller.ID].GamesPlayed)
		a.Equal(2, byPlayer[caller.ID].TablesJoined)
		a.NotEmpty(byPlayer[caller.ID].DisplayName)
	}

	// other: only tableA is in scope, so tableD's +20 is excluded
	if a.Contains(byPlayer, other.ID) {
		a.Equal(-100, byPlayer[other.ID].NetWinnings)
		a.Equal(1, byPlayer[other.ID].GamesPlayed)
		a.Equal(1, byPlayer[other.ID].TablesJoined)
	}

	// ordering among this test's players: caller (150) before other (-100)
	var callerIdx, otherIdx = -1, -1
	for i, e := range entries {
		switch e.PlayerID {
		case caller.ID:
			callerIdx = i
		case other.ID:
			otherIdx = i
		}
	}
	a.True(callerIdx >= 0 && otherIdx >= 0 && callerIdx < otherIdx, "caller should rank above other")

	// soft-delete tableB: caller's total drops to tableA-only figures
	tableB.Deleted = true
	require.NoError(t, testRepos.Tables.Save(cbg, tableB))

	entries, err = testRepos.Players.GetLeaderboard(cbg, caller.ID, from, to)
	a.NoError(err)
	byPlayer = make(map[int64]*LeaderboardEntry)
	for _, e := range entries {
		byPlayer[e.PlayerID] = e
	}
	if a.Contains(byPlayer, caller.ID) {
		a.Equal(100, byPlayer[caller.ID].NetWinnings)
		a.Equal(1, byPlayer[caller.ID].GamesPlayed)
		a.Equal(1, byPlayer[caller.ID].TablesJoined)
	}

	// a future-only window excludes everything
	future := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	futureEnd := time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC)
	entries, err = testRepos.Players.GetLeaderboard(cbg, caller.ID, future, futureEnd)
	a.NoError(err)
	a.Equal(0, len(entries))
}

func TestGetPlayersCount(t *testing.T) {
	a := assert.New(t)

	p := player()

	// numeric search matches exactly one player by id
	count, err := testRepos.Players.GetPlayersCount(cbg, strconv.FormatInt(p.ID, 10))
	a.NoError(err)
	a.Equal(int64(1), count)

	// email prefix (emails are unique) matches exactly one
	count, err = testRepos.Players.GetPlayersCount(cbg, p.Email)
	a.NoError(err)
	a.Equal(int64(1), count)

	// a search matching nothing returns zero
	count, err = testRepos.Players.GetPlayersCount(cbg, "no-such-player-"+p.Email)
	a.NoError(err)
	a.Equal(int64(0), count)

	// empty search counts all players (at least the ones created here)
	count, err = testRepos.Players.GetPlayersCount(cbg, "")
	a.NoError(err)
	a.Greater(count, int64(0))
}

func TestGetTablesCount(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true // to rapidly create tables
	_, err := testRepos.Tables.CreateTable(cbg, p, "Count Table 1")
	require.NoError(t, err)
	t2, err := testRepos.Tables.CreateTable(cbg, p, "Count Table 2")
	require.NoError(t, err)

	count, err := testRepos.Players.GetTablesCount(cbg, p)
	a.NoError(err)
	a.Equal(int64(2), count)

	// soft-deleting a table removes it from the count
	t2.Deleted = true
	require.NoError(t, testRepos.Tables.Save(cbg, t2))

	count, err = testRepos.Players.GetTablesCount(cbg, p)
	a.NoError(err)
	a.Equal(int64(1), count)
}

func TestGetPlayerTablesFilteredCount(t *testing.T) {
	a := assert.New(t)

	// The count is player-scoped, and p is a brand-new player, so a wide date range
	// matches only this test's tables — no created-window isolation needed.
	p := player()
	p.IsSiteAdmin = true // to rapidly create tables
	_, err := testRepos.Tables.CreateTable(cbg, p, "FCount Table 1")
	require.NoError(t, err)
	_, err = testRepos.Tables.CreateTable(cbg, p, "FCount Table 2")
	require.NoError(t, err)
	t3, err := testRepos.Tables.CreateTable(cbg, p, "FCount Table 3")
	require.NoError(t, err)

	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	count, err := testRepos.Players.GetPlayerTablesFilteredCount(cbg, p.ID, from, to)
	a.NoError(err)
	a.Equal(int64(3), count)

	// soft-deleted tables are excluded
	t3.Deleted = true
	require.NoError(t, testRepos.Tables.Save(cbg, t3))
	count, err = testRepos.Players.GetPlayerTablesFilteredCount(cbg, p.ID, from, to)
	a.NoError(err)
	a.Equal(int64(2), count)

	// a future-only window excludes everything (date filter on tables.created)
	future := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	futureEnd := time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC)
	count, err = testRepos.Players.GetPlayerTablesFilteredCount(cbg, p.ID, future, futureEnd)
	a.NoError(err)
	a.Equal(int64(0), count)
}

func TestGetPlayerProfile(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true
	tbl, _ := testRepos.Tables.CreateTable(cbg, p, "Profile Table")

	game, _ := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	_ = testRepos.Games.EndGame(cbg, game, nil, map[int64]int{p.ID: 500})

	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	profile, err := testRepos.Players.GetPlayerProfile(cbg, p.ID, from, to, 0, 100)
	a.NoError(err)
	a.Equal(p.ID, profile.Player.ID)
	a.Equal(1, profile.Stats.TablesJoined)
	a.Equal(1, profile.Stats.GamesPlayed)
	a.Equal(500, profile.Stats.TotalWinnings)
	a.Equal(1, len(profile.Tables))
	a.Equal(1, len(profile.GraphData))
	a.Equal(500, profile.GraphData[0].Balance)

	// Test graph data is not affected by pagination
	profile2, err := testRepos.Players.GetPlayerProfile(cbg, p.ID, from, to, 0, 0)
	a.NoError(err)
	a.Equal(0, len(profile2.Tables))
	a.Equal(1, len(profile2.GraphData))

	// Test with non-existent player
	_, err = testRepos.Players.GetPlayerProfile(cbg, 0, from, to, 0, 100)
	a.Error(err)
}

func TestGetPlayerGraphData(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true
	tbl, _ := testRepos.Tables.CreateTable(cbg, p, "Graph Table 1")

	game, _ := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	_ = testRepos.Games.EndGame(cbg, game, nil, map[int64]int{p.ID: 300})

	tbl2, _ := testRepos.Tables.CreateTable(cbg, p, "Graph Table 2")
	game2, _ := testRepos.Games.CreateGame(cbg, tbl2, "bourre")
	_ = testRepos.Games.EndGame(cbg, game2, nil, map[int64]int{p.ID: -100})

	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	points, err := testRepos.Players.GetPlayerGraphData(cbg, p.ID, from, to)
	a.NoError(err)
	a.Equal(2, len(points))
	a.Equal(300, points[0].Balance)
	a.Equal(-100, points[1].Balance)

	// Test with filtered date range that excludes all data
	future := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	points2, err := testRepos.Players.GetPlayerGraphData(cbg, p.ID, future, to)
	a.NoError(err)
	a.Equal(0, len(points2))
}

func TestPlayer_Delete(t *testing.T) {
	p := player()
	email := p.Email
	displayName := p.DisplayName

	a := assert.New(t)
	a.NoError(testRepos.Players.Delete(cbg, p))
	a.NotEqual(email, p.Email)
	a.NotEqual(displayName, p.DisplayName)

	oldRecord, _ := testRepos.Players.GetPlayerByEmail(cbg, email)
	a.Nil(oldRecord)

	newRecord, _ := testRepos.Players.GetPlayerByEmail(cbg, p.Email)
	a.NotEqual(email, newRecord.Email)
	a.NotEqual(displayName, newRecord.DisplayName)
}
