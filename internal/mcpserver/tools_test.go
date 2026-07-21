package mcpserver

import (
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"mondaynightpoker-server/pkg/room/gamefactory"
)

func ptrInt64(v int64) *int64 { return &v }
func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }

func TestListPlayers(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	p := createPlayer(t)

	// search by exact id returns the player
	_, out, err := s.listPlayers(cbg, nil, listPlayersInput{Search: ptrStr(strconv.FormatInt(p.ID, 10))})
	a.NoError(err)
	a.Len(out.Players, 1)
	a.Equal(p.ID, out.Players[0].ID)
	a.Equal(p.Email, out.Players[0].Email)

	// unfiltered list returns at least the created player
	_, out, err = s.listPlayers(cbg, nil, listPlayersInput{})
	a.NoError(err)
	a.NotEmpty(out.Players)
}

func TestListPlayers_PaginationClamp(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	// rows > 100 clamps to 100 (no error)
	_, out, err := s.listPlayers(cbg, nil, listPlayersInput{Rows: ptrInt(5000)})
	a.NoError(err)
	a.LessOrEqual(len(out.Players), maxRows)

	// rows <= 0 clamps to the minimum
	_, _, err = s.listPlayers(cbg, nil, listPlayersInput{Rows: ptrInt(0)})
	a.NoError(err)

	// negative start errors
	_, _, err = s.listPlayers(cbg, nil, listPlayersInput{Start: ptrInt64(-1)})
	a.Error(err)
}

func TestGetPlayer(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	p := createPlayer(t)

	_, out, err := s.getPlayer(cbg, nil, getPlayerInput{ID: p.ID})
	a.NoError(err)
	a.Equal(p.ID, out.ID)
	a.Equal(p.DisplayName, out.DisplayName)
	a.Equal(string(p.Status), out.Status)

	// not found
	_, _, err = s.getPlayer(cbg, nil, getPlayerInput{ID: -999})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestGetPlayerByEmail(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	p := createPlayer(t)

	_, out, err := s.getPlayerByEmail(cbg, nil, getPlayerByEmailInput{Email: p.Email})
	a.NoError(err)
	a.Equal(p.ID, out.ID)

	// not found
	_, _, err = s.getPlayerByEmail(cbg, nil, getPlayerByEmailInput{Email: "does-not-exist@example.domain"})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestGetPlayerStats(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Stats Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "Bourré")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 100}))

	// default (wide) date range picks up the activity
	_, out, err := s.getPlayerStats(cbg, nil, getPlayerStatsInput{ID: admin.ID})
	a.NoError(err)
	a.Equal(1, out.TablesJoined)
	a.Equal(1, out.GamesPlayed)
	a.Equal(100, out.TotalWinnings)

	// bad date input errors
	_, _, err = s.getPlayerStats(cbg, nil, getPlayerStatsInput{ID: admin.ID, From: ptrStr("not-a-date")})
	a.Error(err)
}

func TestGetPlayerProfile(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Profile Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 500}))

	_, out, err := s.getPlayerProfile(cbg, nil, getPlayerProfileInput{ID: admin.ID})
	a.NoError(err)
	a.Equal(admin.ID, out.Player.ID)
	a.NotEmpty(out.Tables)
	a.Equal(500, out.Stats.TotalWinnings)

	// not found
	_, _, err = s.getPlayerProfile(cbg, nil, getPlayerProfileInput{ID: -999})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestListPlayerTables_Unfiltered(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Unfiltered Table")
	a.NoError(err)

	_, out, err := s.listPlayerTables(cbg, nil, listPlayerTablesInput{ID: admin.ID})
	a.NoError(err)
	a.NotEmpty(out.Tables)

	found := false
	for _, tw := range out.Tables {
		if tw.UUID == tbl.UUID {
			found = true
		}
	}
	a.True(found)

	// not found (unfiltered path fetches the player first)
	_, _, err = s.listPlayerTables(cbg, nil, listPlayerTablesInput{ID: -999})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestListPlayerTables_Filtered(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	_, err := testRepos.Tables.CreateTable(cbg, admin, "Filtered Table")
	a.NoError(err)

	// wide date range picks up the table
	_, out, err := s.listPlayerTables(cbg, nil, listPlayerTablesInput{
		ID:   admin.ID,
		From: ptrStr("2000-01-01T00:00:00Z"),
		To:   ptrStr("2100-01-01T00:00:00Z"),
	})
	a.NoError(err)
	a.NotEmpty(out.Tables)

	// narrow (future) date range excludes everything
	_, out, err = s.listPlayerTables(cbg, nil, listPlayerTablesInput{
		ID:   admin.ID,
		From: ptrStr("2100-01-01T00:00:00Z"),
		To:   ptrStr("2100-12-31T00:00:00Z"),
	})
	a.NoError(err)
	a.Empty(out.Tables)

	// bad date input errors
	_, _, err = s.listPlayerTables(cbg, nil, listPlayerTablesInput{ID: admin.ID, From: ptrStr("nope")})
	a.Error(err)
}

func TestListTables(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Listed Table")
	a.NoError(err)

	_, out, err := s.listTables(cbg, nil, listTablesInput{})
	a.NoError(err)
	a.NotEmpty(out.Tables)

	found := false
	for _, tw := range out.Tables {
		if tw.UUID == tbl.UUID {
			found = true
			a.Equal(admin.Email, tw.PlayerEmail)
		}
	}
	a.True(found)

	// rows > 100 clamps
	_, out, err = s.listTables(cbg, nil, listTablesInput{Rows: ptrInt(9999)})
	a.NoError(err)
	a.LessOrEqual(len(out.Tables), maxRows)

	// negative start errors
	_, _, err = s.listTables(cbg, nil, listTablesInput{Start: ptrInt64(-5)})
	a.Error(err)
}

func TestGetTable(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Get Table")
	a.NoError(err)

	_, out, err := s.getTable(cbg, nil, getTableInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Equal(tbl.UUID, out.Table.UUID)
	a.Equal("Get Table", out.Table.Name)
	a.Equal(int64(0), out.GamesCount)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NotNil(game)

	_, out, err = s.getTable(cbg, nil, getTableInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Equal(int64(1), out.GamesCount)

	// not found (valid uuid syntax that does not exist)
	_, _, err = s.getTable(cbg, nil, getTableInput{UUID: uuid.New().String()})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestGetTableRoster(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Roster Table")
	a.NoError(err)

	other := createPlayer(t)
	_, err = testRepos.Tables.Join(cbg, other, tbl)
	a.NoError(err)

	_, out, err := s.getTableRoster(cbg, nil, getTableRosterInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Equal(tbl.UUID, out.Table.UUID)
	a.Len(out.Players, 2)

	// the creator should be a table admin
	var adminEntry *PlayerTableDTO
	for i := range out.Players {
		if out.Players[i].PlayerID == admin.ID {
			adminEntry = &out.Players[i]
		}
	}
	a.NotNil(adminEntry)
	a.True(adminEntry.IsTableAdmin)
	a.Equal(admin.Email, adminEntry.Player.Email)

	// not found (valid uuid syntax that does not exist)
	_, _, err = s.getTableRoster(cbg, nil, getTableRosterInput{UUID: uuid.New().String()})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestListGameTypes(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	_, out, err := s.listGameTypes(cbg, nil, listGameTypesInput{})
	a.NoError(err)
	a.Equal(gamefactory.Names(), out.GameTypes)
	a.Contains(out.GameTypes, "bourre")
}
