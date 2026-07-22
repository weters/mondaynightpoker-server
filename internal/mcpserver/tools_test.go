package mcpserver

import (
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mondaynightpoker-server/pkg/room/gamefactory"
)

func ptrInt64(v int64) *int64 { return &v }
func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }

// -------------------- list_players (data + email visibility) --------------------

func TestListPlayers_Data(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	caller := adminCaller(admin.ID)

	p := createPlayer(t)

	// search by exact id returns the player, email visible to the admin
	out, err := s.listPlayers(cbg, nil, caller, listPlayersInput{Search: ptrStr(strconv.FormatInt(p.ID, 10))})
	a.NoError(err)
	a.Len(out.Players, 1)
	a.Equal(p.ID, out.Players[0].ID)
	require.NotNil(t, out.Players[0].Email)
	a.Equal(p.Email, *out.Players[0].Email)

	// pagination clamps still hold
	out, err = s.listPlayers(cbg, nil, caller, listPlayersInput{Rows: ptrInt(5000)})
	a.NoError(err)
	a.LessOrEqual(len(out.Players), maxRows)

	_, err = s.listPlayers(cbg, nil, caller, listPlayersInput{Start: ptrInt64(-1)})
	a.Error(err)
}

// -------------------- get_player (data + email visibility) --------------------

func TestGetPlayer_Data(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	p := createPlayer(t)

	// admin sees any player's email
	out, err := s.getPlayer(cbg, nil, adminCaller(admin.ID), getPlayerInput{ID: p.ID})
	a.NoError(err)
	a.Equal(p.ID, out.ID)
	require.NotNil(t, out.Email)
	a.Equal(p.Email, *out.Email)

	// a caller always sees their own email
	out, err = s.getPlayer(cbg, nil, playerCaller(p.ID), getPlayerInput{ID: p.ID})
	a.NoError(err)
	a.Equal(p.ID, out.ID)
	require.NotNil(t, out.Email)
	a.Equal(p.Email, *out.Email)

	// not found
	_, err = s.getPlayer(cbg, nil, adminCaller(admin.ID), getPlayerInput{ID: -999})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

// -------------------- get_player_by_email (data) --------------------

func TestGetPlayerByEmail_Data(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	p := createPlayer(t)

	out, err := s.getPlayerByEmail(cbg, nil, adminCaller(admin.ID), getPlayerByEmailInput{Email: p.Email})
	a.NoError(err)
	a.Equal(p.ID, out.ID)
	require.NotNil(t, out.Email)
	a.Equal(p.Email, *out.Email)

	_, err = s.getPlayerByEmail(cbg, nil, adminCaller(admin.ID), getPlayerByEmailInput{Email: "does-not-exist@example.domain"})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

// -------------------- get_player_stats (data) --------------------

func TestGetPlayerStats(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Stats Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "Bourré")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 100}))

	out, err := s.getPlayerStats(cbg, nil, adminCaller(admin.ID), getPlayerStatsInput{ID: admin.ID})
	a.NoError(err)
	a.Equal(1, out.TablesJoined)
	a.Equal(1, out.GamesPlayed)
	a.Equal(100, out.TotalWinnings)

	// bad date input errors
	_, err = s.getPlayerStats(cbg, nil, adminCaller(admin.ID), getPlayerStatsInput{ID: admin.ID, From: ptrStr("not-a-date")})
	a.Error(err)
}

// -------------------- get_player_profile (data + email visibility) --------------------

func TestGetPlayerProfile(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Profile Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 500}))

	out, err := s.getPlayerProfile(cbg, nil, adminCaller(admin.ID), getPlayerProfileInput{ID: admin.ID})
	a.NoError(err)
	a.Equal(admin.ID, out.Player.ID)
	a.NotEmpty(out.Tables)
	a.Equal(500, out.Stats.TotalWinnings)
	// self/admin can see the profile's own email
	require.NotNil(t, out.Player.Email)
	a.Equal(admin.Email, *out.Player.Email)

	// not found (admin path)
	_, err = s.getPlayerProfile(cbg, nil, adminCaller(admin.ID), getPlayerProfileInput{ID: -999})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

// -------------------- list_player_tables (data) --------------------

func TestListPlayerTables_Unfiltered(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Unfiltered Table")
	a.NoError(err)

	out, err := s.listPlayerTables(cbg, nil, adminCaller(admin.ID), listPlayerTablesInput{ID: admin.ID})
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
	_, err = s.listPlayerTables(cbg, nil, adminCaller(admin.ID), listPlayerTablesInput{ID: -999})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestListPlayerTables_Filtered(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	_, err := testRepos.Tables.CreateTable(cbg, admin, "Filtered Table")
	a.NoError(err)
	caller := adminCaller(admin.ID)

	// wide date range picks up the table
	out, err := s.listPlayerTables(cbg, nil, caller, listPlayerTablesInput{
		ID:   admin.ID,
		From: ptrStr("2000-01-01T00:00:00Z"),
		To:   ptrStr("2100-01-01T00:00:00Z"),
	})
	a.NoError(err)
	a.NotEmpty(out.Tables)

	// narrow (future) date range excludes everything
	out, err = s.listPlayerTables(cbg, nil, caller, listPlayerTablesInput{
		ID:   admin.ID,
		From: ptrStr("2100-01-01T00:00:00Z"),
		To:   ptrStr("2100-12-31T00:00:00Z"),
	})
	a.NoError(err)
	a.Empty(out.Tables)

	// bad date input errors
	_, err = s.listPlayerTables(cbg, nil, caller, listPlayerTablesInput{ID: admin.ID, From: ptrStr("nope")})
	a.Error(err)
}

// -------------------- list_tables (admin: all + email; non-admin: membership, no email) --------------------

func TestListTables_Admin(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Listed Table")
	a.NoError(err)
	caller := adminCaller(admin.ID)

	out, err := s.listTables(cbg, nil, caller, listTablesInput{})
	a.NoError(err)
	a.NotEmpty(out.Tables)

	found := false
	for _, tw := range out.Tables {
		if tw.UUID == tbl.UUID {
			found = true
			require.NotNil(t, tw.PlayerEmail)
			a.Equal(admin.Email, *tw.PlayerEmail)
		}
	}
	a.True(found)

	// rows > 100 clamps
	out, err = s.listTables(cbg, nil, caller, listTablesInput{Rows: ptrInt(9999)})
	a.NoError(err)
	a.LessOrEqual(len(out.Tables), maxRows)

	// negative start errors
	_, err = s.listTables(cbg, nil, caller, listTablesInput{Start: ptrInt64(-5)})
	a.Error(err)
}

func TestListTables_NonAdminMembershipOnly(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	mine, err := testRepos.Tables.CreateTable(cbg, admin, "Mine Table")
	a.NoError(err)
	theirs, err := testRepos.Tables.CreateTable(cbg, admin, "Not Mine Table")
	a.NoError(err)

	member := createPlayer(t)
	_, err = testRepos.Tables.Join(cbg, member, mine)
	a.NoError(err)

	// the handler scopes the result set by the passed-in caller's membership (data-scoping).
	out, err := s.listTables(cbg, nil, playerCaller(member.ID), listTablesInput{})
	a.NoError(err)

	var sawMine bool
	for _, tw := range out.Tables {
		// only tables the caller belongs to are returned
		a.NotEqual(theirs.UUID, tw.UUID)
		// creator emails are absent (nil, omitted from JSON) for non-admins
		a.Nil(tw.PlayerEmail)
		if tw.UUID == mine.UUID {
			sawMine = true
		}
	}
	a.True(sawMine)
}

// -------------------- get_table (data) --------------------

func TestGetTable(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Get Table")
	a.NoError(err)

	// any authenticated caller may fetch any table by uuid
	outsider := createPlayer(t)
	out, err := s.getTable(cbg, nil, playerCaller(outsider.ID), getTableInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Equal(tbl.UUID, out.Table.UUID)
	a.Equal("Get Table", out.Table.Name)
	a.Equal(int64(0), out.GamesCount)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NotNil(game)

	out, err = s.getTable(cbg, nil, adminCaller(admin.ID), getTableInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Equal(int64(1), out.GamesCount)

	// not found (valid uuid syntax that does not exist)
	_, err = s.getTable(cbg, nil, playerCaller(outsider.ID), getTableInput{UUID: uuid.New().String()})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

// -------------------- get_table_roster (email visibility) --------------------

func TestGetTableRoster_Admin(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Roster Table")
	a.NoError(err)

	other := createPlayer(t)
	_, err = testRepos.Tables.Join(cbg, other, tbl)
	a.NoError(err)

	out, err := s.getTableRoster(cbg, nil, adminCaller(admin.ID), getTableRosterInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Equal(tbl.UUID, out.Table.UUID)
	a.Len(out.Players, 2)

	// the admin sees every email
	for _, pt := range out.Players {
		if pt.PlayerID == admin.ID {
			a.True(pt.IsTableAdmin)
			require.NotNil(t, pt.Player.Email)
			a.Equal(admin.Email, *pt.Player.Email)
		}
		if pt.PlayerID == other.ID {
			require.NotNil(t, pt.Player.Email)
			a.Equal(other.Email, *pt.Player.Email)
		}
	}

	// not found (valid uuid syntax that does not exist)
	_, err = s.getTableRoster(cbg, nil, adminCaller(admin.ID), getTableRosterInput{UUID: uuid.New().String()})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestGetTableRoster_NonAdminRedaction(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Roster Redact Table")
	a.NoError(err)

	member := createPlayer(t)
	_, err = testRepos.Tables.Join(cbg, member, tbl)
	a.NoError(err)

	out, err := s.getTableRoster(cbg, nil, playerCaller(member.ID), getTableRosterInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Len(out.Players, 2)

	for _, pt := range out.Players {
		switch pt.PlayerID {
		case member.ID:
			// the caller keeps their own email
			require.NotNil(t, pt.Player.Email)
			a.Equal(member.Email, *pt.Player.Email)
		case admin.ID:
			// every other member's email is absent (nil, omitted from JSON)
			a.Nil(pt.Player.Email)
		}
	}
}

// -------------------- list_game_types (data) --------------------

func TestListGameTypes(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	out, err := s.listGameTypes(cbg, nil, playerCaller(1), listGameTypesInput{})
	a.NoError(err)
	a.Equal(gamefactory.Names(), out.GameTypes)
	a.Contains(out.GameTypes, "bourre")
}
