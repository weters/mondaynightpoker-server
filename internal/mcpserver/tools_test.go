package mcpserver

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/money"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/room/gamefactory"
)

func ptrInt64(v int64) *int64 { return &v }
func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }
func ptrBool(v bool) *bool    { return &v }

// -------------------- whoami (data) --------------------

func TestWhoami_Data(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	p := createPlayer(t)

	// the caller always gets their own record back, email included
	out, err := s.whoami(cbg, nil, playerCaller(p.ID), whoamiInput{})
	a.NoError(err)
	a.Equal(p.ID, out.ID)
	a.Equal(p.DisplayName, out.DisplayName)
	a.False(out.IsSiteAdmin)
	require.NotNil(t, out.Email)
	a.Equal(p.Email, *out.Email)

	// an admin caller gets their own record, not somebody else's
	admin := createSiteAdmin(t)
	out, err = s.whoami(cbg, nil, adminCaller(admin.ID), whoamiInput{})
	a.NoError(err)
	a.Equal(admin.ID, out.ID)
	a.True(out.IsSiteAdmin)
	require.NotNil(t, out.Email)
	a.Equal(admin.Email, *out.Email)

	// the record is read live, so the caller's claimed admin bit does not decide the
	// reported one: a stale token claiming admin still reports the stored value
	out, err = s.whoami(cbg, nil, adminCaller(p.ID), whoamiInput{})
	a.NoError(err)
	a.Equal(p.ID, out.ID)
	a.False(out.IsSiteAdmin)

	// a caller whose player record no longer exists is reported as absent
	_, err = s.whoami(cbg, nil, playerCaller(-999), whoamiInput{})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

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
	a.Equal(100, out.TotalWinningsCents)
	a.Equal("$1", out.TotalWinningsDisplay)

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
	a.Equal(500, out.Stats.TotalWinningsCents)
	a.Equal("$5", out.Stats.TotalWinningsDisplay)
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
	// aggregates: the creator is seated, so playersCount is at least one and the total
	// balance carries its preformatted display twin
	a.GreaterOrEqual(out.PlayersCount, 1)
	a.Equal(money.FormatCents(out.TotalBalanceCents), out.TotalBalanceDisplay)

	// not found (valid uuid syntax that does not exist)
	_, err = s.getTable(cbg, nil, playerCaller(outsider.ID), getTableInput{UUID: uuid.New().String()})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestGetTable_DeletedIsNotFound(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Deleted Get Table")
	a.NoError(err)

	// while live, the table is retrievable
	_, err = s.getTable(cbg, nil, adminCaller(admin.ID), getTableInput{UUID: tbl.UUID})
	a.NoError(err)

	// once soft-deleted, the tool reports it as not found (even to a site admin)
	tbl.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, tbl))

	_, err = s.getTable(cbg, nil, adminCaller(admin.ID), getTableInput{UUID: tbl.UUID})
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

func TestGetTableRoster_DeletedIsNotFound(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Deleted Roster Table")
	a.NoError(err)

	tbl.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, tbl))

	_, err = s.getTableRoster(cbg, nil, adminCaller(admin.ID), getTableRosterInput{UUID: tbl.UUID})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

func TestListTables_ExcludesDeleted(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	live, err := testRepos.Tables.CreateTable(cbg, admin, "Live Listed Table")
	a.NoError(err)
	gone, err := testRepos.Tables.CreateTable(cbg, admin, "Deleted Listed Table")
	a.NoError(err)

	gone.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, gone))

	// admin path: the deleted table must not be listed, the live one must be
	out, err := s.listTables(cbg, nil, adminCaller(admin.ID), listTablesInput{Rows: ptrInt(maxRows)})
	a.NoError(err)

	var sawLive, sawDeleted bool
	for _, tw := range out.Tables {
		switch tw.UUID {
		case live.UUID:
			sawLive = true
		case gone.UUID:
			sawDeleted = true
		}
	}
	a.True(sawLive, "expected the live table to be listed")
	a.False(sawDeleted, "did not expect the deleted table to be listed")
}

// -------------------- deleted-table tripwire --------------------

// TestTools_NoDirectTableLookup is a source-level tripwire. TableRepo.GetTableByUUID
// returns soft-deleted rows, so a tool that calls it directly has to remember the
// Deleted check; activeTable is the one door that cannot be forgotten. Behavioral
// tests only cover the tools that exist today, and the original miss here was a
// surface nobody thought to test, so this asserts the rule for tools not yet written.
func TestTools_NoDirectTableLookup(t *testing.T) {
	src, err := os.ReadFile("tools.go")
	require.NoError(t, err)

	assert.NotContains(t, string(src), "GetTableByUUID",
		"tool handlers must resolve a uuid through s.activeTable, which rejects soft-deleted "+
			"tables; GetTableByUUID returns them and is reserved for the admin API")
}

// -------------------- list_game_types (data) --------------------

func TestListGameTypes(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	out, err := s.listGameTypes(cbg, nil, playerCaller(1), listGameTypesInput{})
	a.NoError(err)

	// one entry per registered game type, in gamefactory.Names() order
	names := gamefactory.Names()
	a.Len(out.GameTypes, len(names))

	byID := make(map[string]string, len(out.GameTypes))
	for i, gt := range out.GameTypes {
		a.Equal(names[i], gt.ID)
		a.NotEmpty(gt.DisplayGroup, "every game type must carry a display group")
		byID[gt.ID] = gt.DisplayGroup
	}

	a.Equal("Bourre", byID["bourre"])
	a.Equal("Seven Card", byID["seven-card"])
	a.Equal("Texas Hold'em", byID["texas-hold-em"])
}

// TestListGameTypes_DisplayGroupExhaustive is a tripwire: for every gamefactory id,
// the display group derived from the factory's DisplayName must match what
// model.GameTypeGroup produces for the game's real display names (the strings
// Details writes into the games.game_type column). A newly registered game type
// whose DisplayName drifts from its stored display names fails here.
func TestListGameTypes_DisplayGroupExhaustive(t *testing.T) {
	a := assert.New(t)

	// representative Details inputs per gamefactory id, covering each game's variants
	detailsInputs := map[string][]playable.AdditionalData{
		"bourre": {
			{},
			{"fiveSuit": true},
		},
		"seven-card": {
			{},
			{"variant": "stud"},
			{"variant": "low-card-wild"},
			{"variant": "baseball"},
			{"variant": "follow-the-queen"},
			{"variant": "high-chicago"},
			{"variant": "chiggs"},
			{"variant": "coupons-and-clippings"},
		},
		"pass-the-poop": {
			{"ante": float64(25), "edition": "standard"},
			{"ante": float64(25), "edition": "diarrhea"},
			{"ante": float64(25), "edition": "pairs", "allowBlocks": true},
		},
		"little-l": {
			{},
			{"initialDeal": float64(5), "tradeIns": []float64{0, 1, 3}},
		},
		"acey-deucey": {
			{},
			{"gameType": "continuous shoe"},
			{"gameType": "chaos"},
			{"allowPass": true},
		},
		"texas-hold-em": {
			{},
			{"variant": "pineapple"},
			{"variant": "lazy-pineapple"},
		},
		"guts": {
			{},
			{"cardCount": float64(3)},
			{"bloodyGuts": true, "allowTrades": true},
		},
	}

	for _, name := range gamefactory.Names() {
		factory, err := gamefactory.Get(name)
		a.NoError(err)

		derived := model.GameTypeGroup(factory.DisplayName())
		a.NotEmptyf(derived, "gamefactory id %q derives an empty display group", name)

		inputs, ok := detailsInputs[name]
		a.Truef(ok, "gamefactory id %q has no Details inputs in this test — add them", name)

		for _, in := range inputs {
			displayName, _, err := factory.Details(in)
			a.NoErrorf(err, "gamefactory id %q: Details(%v) failed", name, in)
			a.Equalf(derived, model.GameTypeGroup(displayName),
				"gamefactory id %q: real display name %q groups to %q, but DisplayName %q derives %q",
				name, displayName, model.GameTypeGroup(displayName), factory.DisplayName(), derived)
		}
	}
}

// -------------------- list_table_games --------------------

func TestListTableGames(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Games Table")
	a.NoError(err)

	other := createPlayer(t)
	_, err = testRepos.Tables.Join(cbg, other, tbl)
	a.NoError(err)

	// no games yet
	out, err := s.listTableGames(cbg, nil, playerCaller(other.ID), listTableGamesInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Empty(out.Games)
	a.Equal(int64(0), out.Total)
	a.False(out.HasMore)

	// play a game with an adjustment for each seated player
	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 300, other.ID: -300}))

	out, err = s.listTableGames(cbg, nil, playerCaller(other.ID), listTableGamesInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Len(out.Games, 1)
	a.Equal(int64(1), out.Total)
	a.False(out.HasMore)

	g := out.Games[0]
	a.Equal(game.ID, g.ID)
	a.Equal("bourre", g.GameType)
	a.Equal("Bourre", g.GameTypeGroup)
	a.Nil(g.ParentID)
	a.NotNil(g.Ended, "an ended game exposes its ended timestamp")

	// adjustments are biggest-winner-first and carry display twins
	a.Len(g.Adjustments, 2)
	a.Equal(admin.ID, g.Adjustments[0].PlayerID)
	a.Equal(300, g.Adjustments[0].AdjustmentCents)
	a.Equal(money.FormatCents(300), g.Adjustments[0].AdjustmentDisplay)
	a.Equal(other.ID, g.Adjustments[1].PlayerID)
	a.Equal(-300, g.Adjustments[1].AdjustmentCents)
	a.Equal(money.FormatCents(-300), g.Adjustments[1].AdjustmentDisplay)
}

func TestListTableGames_Pagination(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Paged Games Table")
	a.NoError(err)

	for i := 0; i < 3; i++ {
		game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
		a.NoError(err)
		a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 100}))
	}

	// first page of two: total is three, more remain
	out, err := s.listTableGames(cbg, nil, adminCaller(admin.ID), listTableGamesInput{UUID: tbl.UUID, Rows: ptrInt(2)})
	a.NoError(err)
	a.Len(out.Games, 2)
	a.Equal(int64(3), out.Total)
	a.True(out.HasMore)

	// last page: no more remain
	out, err = s.listTableGames(cbg, nil, adminCaller(admin.ID), listTableGamesInput{UUID: tbl.UUID, Start: ptrInt64(2), Rows: ptrInt(2)})
	a.NoError(err)
	a.Len(out.Games, 1)
	a.Equal(int64(3), out.Total)
	a.False(out.HasMore)
}

func TestListTableGames_DeletedIsNotFound(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Deleted Games Table")
	a.NoError(err)

	tbl.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, tbl))

	_, err = s.listTableGames(cbg, nil, adminCaller(admin.ID), listTableGamesInput{UUID: tbl.UUID})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

// -------------------- get_game --------------------

func TestGetGame(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Get Game Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, map[string]interface{}{"log": "hello"}, map[int64]int{admin.ID: 250}))

	// default: no log, but summary + adjustments present. The table uuid is the
	// capability, so any authenticated player holding it may read the game.
	outsider := createPlayer(t)
	out, err := s.getGame(cbg, nil, playerCaller(outsider.ID), getGameInput{UUID: tbl.UUID, ID: game.ID})
	a.NoError(err)
	a.Equal(game.ID, out.ID)
	a.Equal("bourre", out.GameType)
	a.Equal("Bourre", out.GameTypeGroup)
	a.Nil(out.Log, "log must be omitted unless includeLog is true")
	a.Len(out.Adjustments, 1)
	a.Equal(250, out.Adjustments[0].AdjustmentCents)
	a.Equal(money.FormatCents(250), out.Adjustments[0].AdjustmentDisplay)

	// includeLog surfaces the stored log payload
	out, err = s.getGame(cbg, nil, playerCaller(outsider.ID), getGameInput{UUID: tbl.UUID, ID: game.ID, IncludeLog: ptrBool(true)})
	a.NoError(err)
	a.NotNil(out.Log)

	// unknown id (with a valid table uuid) is not found
	_, err = s.getGame(cbg, nil, adminCaller(admin.ID), getGameInput{UUID: tbl.UUID, ID: -999})
	a.Error(err)
	a.Contains(err.Error(), "game not found")

	// unknown table uuid is not found (never "table not found": don't reveal the game exists)
	_, err = s.getGame(cbg, nil, adminCaller(admin.ID), getGameInput{UUID: uuid.New().String(), ID: game.ID})
	a.Error(err)
	a.Contains(err.Error(), "game not found")
}

// TestGetGame_WrongTableIsGameNotFound is the enumeration guard: a valid game id
// requested with a different (valid, non-deleted) table's uuid must not leak the
// game. Because games.id is sequential, keying on id alone would let any caller walk
// the id space and read every game site-wide; the uuid is the capability.
func TestGetGame_WrongTableIsGameNotFound(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)

	owner, err := testRepos.Tables.CreateTable(cbg, admin, "Owner Table")
	a.NoError(err)
	game, err := testRepos.Games.CreateGame(cbg, owner, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 100}))

	// a second, unrelated live table the caller might legitimately hold a uuid for
	other, err := testRepos.Tables.CreateTable(cbg, admin, "Other Table")
	a.NoError(err)

	// the game is real and the uuid is real, but they do not match: game not found
	_, err = s.getGame(cbg, nil, adminCaller(admin.ID), getGameInput{UUID: other.UUID, ID: game.ID})
	a.Error(err)
	a.Contains(err.Error(), "game not found")
	a.NotContains(err.Error(), "table")

	// sanity: the same id resolves fine against its own table's uuid
	out, err := s.getGame(cbg, nil, adminCaller(admin.ID), getGameInput{UUID: owner.UUID, ID: game.ID})
	a.NoError(err)
	a.Equal(game.ID, out.ID)
}

// TestGetGame_GameTypeGroupFromDisplayName exercises the GameSummaryDTO group mapping
// with a realistic game_type display name (not a gamefactory slug), so the mapping is
// verified through model.GameTypeGroup rather than a prefix coincidence with "bourre".
func TestGetGame_GameTypeGroupFromDisplayName(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Display Name Game Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "4-Card Little L (trade: 0, 2)")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 100}))

	out, err := s.getGame(cbg, nil, adminCaller(admin.ID), getGameInput{UUID: tbl.UUID, ID: game.ID})
	a.NoError(err)
	a.Equal("4-Card Little L (trade: 0, 2)", out.GameType)
	a.Equal("Little L", out.GameTypeGroup)
}

func TestGetGame_DeletedTableIsGameNotFound(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Deleted Get Game Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 100}))

	tbl.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, tbl))

	// the game exists, but its table is gone: report the GAME as missing, never the
	// table (which would leak that the game exists at a deleted table)
	_, err = s.getGame(cbg, nil, adminCaller(admin.ID), getGameInput{UUID: tbl.UUID, ID: game.ID})
	a.Error(err)
	a.Contains(err.Error(), "game not found")
	a.NotContains(err.Error(), "table")
}

// -------------------- get_table_stats --------------------

func TestGetTableStats(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Stats Roster Table")
	a.NoError(err)

	loser := createPlayer(t)
	_, err = testRepos.Tables.Join(cbg, loser, tbl)
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 400, loser.ID: -400}))

	// any authenticated player may read the stats; no emails are exposed
	out, err := s.getTableStats(cbg, nil, playerCaller(loser.ID), getTableStatsInput{UUID: tbl.UUID})
	a.NoError(err)
	a.Equal(tbl.UUID, out.Table.UUID)
	a.Len(out.Players, 2)

	// sorted by net winnings descending: the winner is first
	winner := out.Players[0]
	a.Equal(admin.ID, winner.PlayerID)
	a.Equal(1, winner.GamesPlayed)
	a.Equal(400, winner.NetWinningsCents)
	a.Equal(money.FormatCents(400), winner.NetWinningsDisplay)
	a.Equal(money.FormatCents(winner.BalanceCents), winner.BalanceDisplay)

	last := out.Players[1]
	a.Equal(loser.ID, last.PlayerID)
	a.Equal(-400, last.NetWinningsCents)
}

func TestGetTableStats_DeletedIsNotFound(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Deleted Stats Table")
	a.NoError(err)

	tbl.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, tbl))

	_, err = s.getTableStats(cbg, nil, adminCaller(admin.ID), getTableStatsInput{UUID: tbl.UUID})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

// -------------------- list_player_transactions --------------------

func TestListPlayerTransactions(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Ledger Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 700}))

	out, err := s.listPlayerTransactions(cbg, nil, adminCaller(admin.ID), listPlayerTransactionsInput{ID: admin.ID})
	a.NoError(err)
	a.NotEmpty(out.Transactions)
	a.Positive(out.Total)

	// find the game-driven transaction and check its enrichment + display twins
	var found bool
	for _, tx := range out.Transactions {
		if tx.GameID != nil && *tx.GameID == game.ID {
			found = true
			a.Equal(tbl.UUID, tx.TableUUID)
			a.Equal("Ledger Table", tx.TableName)
			require.NotNil(t, tx.GameType)
			a.Equal("bourre", *tx.GameType)
			a.Equal(money.FormatCents(tx.AdjustmentCents), tx.AdjustmentDisplay)
			a.Equal(money.FormatCents(tx.PreviousBalanceCents), tx.PreviousBalanceDisplay)
			a.Equal(money.FormatCents(tx.CurrentBalanceCents), tx.CurrentBalanceDisplay)
		}
	}
	a.True(found, "expected a game-driven ledger entry")
}

func TestListPlayerTransactions_TableFilterDeletedIsNotFound(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Ledger Filter Table")
	a.NoError(err)

	// narrowing to a live table works
	_, err = s.listPlayerTransactions(cbg, nil, adminCaller(admin.ID), listPlayerTransactionsInput{ID: admin.ID, TableUUID: ptrStr(tbl.UUID)})
	a.NoError(err)

	// narrowing to a soft-deleted table is a not-found error (resolved via activeTable)
	tbl.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, tbl))

	_, err = s.listPlayerTransactions(cbg, nil, adminCaller(admin.ID), listPlayerTransactionsInput{ID: admin.ID, TableUUID: ptrStr(tbl.UUID)})
	a.Error(err)
	a.Contains(err.Error(), "not found")
}

// TestListPlayerTransactions_SelfScoped verifies the input carries the target player
// id so the accessSelfScoped wrapper can deny a non-admin targeting another player,
// while an admin may target anyone.
func TestListPlayerTransactions_SelfScoped(t *testing.T) {
	a := assert.New(t)

	// the input exposes its target id to the policy wrapper
	a.Equal(int64(42), listPlayerTransactionsInput{ID: 42}.targetPlayerID())

	// non-admin targeting another player is denied before the handler runs
	spy := &spyHandler[listPlayerTransactionsInput]{}
	h := wrapHandler[listPlayerTransactionsInput, struct{}](accessSelfScoped, spy.handle)
	_, _, err := h(ctxForPlayer(5), nil, listPlayerTransactionsInput{ID: 6})
	a.ErrorIs(err, errPermissionDenied)
	a.False(spy.ran)

	// admin may target anyone
	spy = &spyHandler[listPlayerTransactionsInput]{}
	h = wrapHandler[listPlayerTransactionsInput, struct{}](accessSelfScoped, spy.handle)
	_, _, err = h(ctxForAdmin(5), nil, listPlayerTransactionsInput{ID: 6})
	a.NoError(err)
	a.True(spy.ran)
}

// -------------------- leaderboard --------------------

func TestLeaderboard(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Leaderboard Table")
	a.NoError(err)

	rival := createPlayer(t)
	_, err = testRepos.Tables.Join(cbg, rival, tbl)
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 900, rival.ID: -900}))

	// scoped to the caller's own tables
	out, err := s.leaderboard(cbg, nil, adminCaller(admin.ID), leaderboardInput{})
	a.NoError(err)
	a.NotEmpty(out.Entries)

	byID := make(map[int64]LeaderboardEntryDTO, len(out.Entries))
	for _, e := range out.Entries {
		byID[e.PlayerID] = e
	}

	require.Contains(t, byID, admin.ID)
	a.Equal(900, byID[admin.ID].NetWinningsCents)
	a.Equal(money.FormatCents(900), byID[admin.ID].NetWinningsDisplay)
	a.GreaterOrEqual(byID[admin.ID].TablesJoined, 1)
	require.Contains(t, byID, rival.ID)
	a.Equal(-900, byID[rival.ID].NetWinningsCents)

	// a player who shares no table with the caller sees an empty (or self-only) board
	stranger := createPlayer(t)
	out, err = s.leaderboard(cbg, nil, playerCaller(stranger.ID), leaderboardInput{})
	a.NoError(err)
	for _, e := range out.Entries {
		a.NotEqual(admin.ID, e.PlayerID, "the stranger must not see the caller's private table")
	}
}

func TestLeaderboard_ExcludesDeletedTables(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Leaderboard Deleted Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "bourre")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, game, nil, map[int64]int{admin.ID: 1000}))

	// while live, the caller's winnings appear
	out, err := s.leaderboard(cbg, nil, adminCaller(admin.ID), leaderboardInput{
		From: ptrStr("2000-01-01T00:00:00Z"),
		To:   ptrStr("2100-01-01T00:00:00Z"),
	})
	a.NoError(err)
	var before int
	for _, e := range out.Entries {
		if e.PlayerID == admin.ID {
			before = e.NetWinningsCents
		}
	}
	a.Equal(1000, before)

	// once the table is deleted, its rows drop out of the leaderboard
	tbl.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, tbl))

	out, err = s.leaderboard(cbg, nil, adminCaller(admin.ID), leaderboardInput{
		From: ptrStr("2000-01-01T00:00:00Z"),
		To:   ptrStr("2100-01-01T00:00:00Z"),
	})
	a.NoError(err)
	for _, e := range out.Entries {
		if e.PlayerID == admin.ID {
			a.NotEqual(1000, e.NetWinningsCents, "deleted-table winnings must not count")
		}
	}
}

// -------------------- list_players pagination --------------------

func TestListPlayers_Pagination(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	caller := adminCaller(admin.ID)

	// total reflects the full match set; hasMore is true when a page cannot hold it all
	out, err := s.listPlayers(cbg, nil, caller, listPlayersInput{Rows: ptrInt(1)})
	a.NoError(err)
	a.Len(out.Players, 1)
	a.Positive(out.Total)
	if out.Total > 1 {
		a.True(out.HasMore)
	}
}

// -------------------- pageTotal (shared pagination metadata) --------------------

// TestPageTotal exercises the short-page shortcut and the count fallback directly,
// since every list tool's pagination metadata flows through it.
func TestPageTotal(t *testing.T) {
	a := assert.New(t)

	countCalls := 0
	count := func(context.Context) (int64, error) {
		countCalls++
		return 10, nil
	}

	// a short page proves the total without a count query
	total, hasMore, err := pageTotal(cbg, 4, 100, 3, count)
	a.NoError(err)
	a.Equal(int64(7), total)
	a.False(hasMore)
	a.Zero(countCalls)

	// an empty first page proves an empty result set
	total, hasMore, err = pageTotal(cbg, 0, 100, 0, count)
	a.NoError(err)
	a.Zero(total)
	a.False(hasMore)
	a.Zero(countCalls)

	// a full page cannot prove the total; the count runs
	total, hasMore, err = pageTotal(cbg, 0, 5, 5, count)
	a.NoError(err)
	a.Equal(int64(10), total)
	a.True(hasMore)
	a.Equal(1, countCalls)

	// an empty page at a non-zero offset proves nothing; the count runs
	total, hasMore, err = pageTotal(cbg, 50, 5, 0, count)
	a.NoError(err)
	a.Equal(int64(10), total)
	a.False(hasMore)
	a.Equal(2, countCalls)

	// count errors propagate
	boom := errors.New("boom")
	_, _, err = pageTotal(cbg, 0, 5, 5, func(context.Context) (int64, error) { return 0, boom })
	a.ErrorIs(err, boom)
}
