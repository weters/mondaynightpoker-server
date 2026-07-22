package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mondaynightpoker-server/internal/oauth"
	"mondaynightpoker-server/pkg/room/gamefactory"
)

// registerTools registers all read-only tools on the given MCP server. Every tool is
// installed through registerTool with a mandatory access policy; the handlers below contain
// no authorization checks of their own and receive the already-authorized caller.
func (s *server) registerTools(m *mcp.Server) {
	registerTool(s, m, &mcp.Tool{
		Name:        "list_players",
		Description: "List players, optionally filtered by a search string (matches id, display name, or email prefix). Site admin only.",
	}, accessAdminOnly, s.listPlayers)

	registerTool(s, m, &mcp.Tool{
		Name:        "get_player",
		Description: "Get a single player by their numeric id. Non-admin callers may only request their own player id.",
	}, accessSelfScoped, s.getPlayer)

	registerTool(s, m, &mcp.Tool{
		Name:        "get_player_by_email",
		Description: "Get a single player by their email address. Site admin only.",
	}, accessAdminOnly, s.getPlayerByEmail)

	registerTool(s, m, &mcp.Tool{
		Name:        "get_player_stats",
		Description: "Get aggregate stats for a player within an optional date range. Non-admin callers may only request their own player id.",
	}, accessSelfScoped, s.getPlayerStats)

	registerTool(s, m, &mcp.Tool{
		Name:        "get_player_profile",
		Description: "Get the full profile for a player (stats, tables, and graph data) within an optional date range. Non-admin callers may only request their own player id.",
	}, accessSelfScoped, s.getPlayerProfile)

	registerTool(s, m, &mcp.Tool{
		Name:        "list_player_tables",
		Description: "List the tables a player belongs to, optionally filtered by a date range. Non-admin callers may only request their own player id.",
	}, accessSelfScoped, s.listPlayerTables)

	registerTool(s, m, &mcp.Tool{
		Name:        "list_tables",
		Description: "List tables. Site admins see every table including the creator's email; non-admin callers see only the tables they belong to, with creator emails omitted.",
	}, accessAuthenticated, s.listTables)

	registerTool(s, m, &mcp.Tool{
		Name:        "get_table",
		Description: "Get a single table by its uuid, including the number of games played. Available to any authenticated player (capability-URL semantics).",
	}, accessAuthenticated, s.getTable)

	registerTool(s, m, &mcp.Tool{
		Name:        "get_table_roster",
		Description: "Get the roster of players at a table, including their table permissions and balances (in cents, with preformatted dollar strings alongside). Available to any authenticated player; for non-admin callers every roster member's email is omitted except the caller's own.",
	}, accessAuthenticated, s.getTableRoster)

	registerTool(s, m, &mcp.Tool{
		Name:        "list_game_types",
		Description: "List the registered game-type identifiers.",
	}, accessAuthenticated, s.listGameTypes)
}

// listPlayersInput is the input for the list_players tool.
type listPlayersInput struct {
	Search *string `json:"search,omitempty" jsonschema:"optional search string matching id, display name, or email prefix"`
	Start  *int64  `json:"start,omitempty" jsonschema:"pagination offset; defaults to 0 and may not be negative"`
	Rows   *int    `json:"rows,omitempty" jsonschema:"number of rows to return; defaults to 100 and is clamped to [1, 100]"`
}

// listPlayersOutput is the output for the list_players tool.
type listPlayersOutput struct {
	Players []PlayerDTO `json:"players" jsonschema:"the matching players"`
}

func (s *server) listPlayers(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in listPlayersInput) (listPlayersOutput, error) {
	offset, limit, err := parsePagination(in.Start, in.Rows)
	if err != nil {
		return listPlayersOutput{}, err
	}

	search := ""
	if in.Search != nil {
		search = *in.Search
	}

	players, err := s.repos.Players.GetPlayersWithSearch(ctx, search, offset, limit)
	if err != nil {
		return listPlayersOutput{}, err
	}

	return listPlayersOutput{Players: fromPlayers(players, caller)}, nil
}

// getPlayerInput is the input for the get_player tool.
type getPlayerInput struct {
	ID int64 `json:"id" jsonschema:"the player's numeric id"`
}

func (in getPlayerInput) targetPlayerID() int64 { return in.ID }

func (s *server) getPlayer(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in getPlayerInput) (PlayerDTO, error) {
	player, err := s.repos.Players.GetPlayerByID(ctx, in.ID)
	if err != nil {
		return PlayerDTO{}, notFound(err, "player")
	}

	return fromPlayer(player, caller), nil
}

// getPlayerByEmailInput is the input for the get_player_by_email tool.
type getPlayerByEmailInput struct {
	Email string `json:"email" jsonschema:"the player's email address"`
}

func (s *server) getPlayerByEmail(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in getPlayerByEmailInput) (PlayerDTO, error) {
	player, err := s.repos.Players.GetPlayerByEmail(ctx, in.Email)
	if err != nil {
		return PlayerDTO{}, notFound(err, "player")
	}

	return fromPlayer(player, caller), nil
}

// getPlayerStatsInput is the input for the get_player_stats tool.
type getPlayerStatsInput struct {
	ID   int64   `json:"id" jsonschema:"the player's numeric id"`
	From *string `json:"from,omitempty" jsonschema:"optional start of the date range (RFC3339); defaults to the epoch"`
	To   *string `json:"to,omitempty" jsonschema:"optional end of the date range (RFC3339); defaults to now"`
}

func (in getPlayerStatsInput) targetPlayerID() int64 { return in.ID }

func (s *server) getPlayerStats(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in getPlayerStatsInput) (PlayerStatsDTO, error) {
	from, to, err := parseDateRange(in.From, in.To)
	if err != nil {
		return PlayerStatsDTO{}, err
	}

	stats, err := s.repos.Players.GetPlayerStats(ctx, in.ID, from, to)
	if err != nil {
		return PlayerStatsDTO{}, notFound(err, "player")
	}

	return fromPlayerStats(stats), nil
}

// getPlayerProfileInput is the input for the get_player_profile tool.
type getPlayerProfileInput struct {
	ID    int64   `json:"id" jsonschema:"the player's numeric id"`
	From  *string `json:"from,omitempty" jsonschema:"optional start of the date range (RFC3339); defaults to the epoch"`
	To    *string `json:"to,omitempty" jsonschema:"optional end of the date range (RFC3339); defaults to now"`
	Start *int64  `json:"start,omitempty" jsonschema:"pagination offset for tables; defaults to 0 and may not be negative"`
	Rows  *int    `json:"rows,omitempty" jsonschema:"number of tables to return; defaults to 100 and is clamped to [1, 100]"`
}

func (in getPlayerProfileInput) targetPlayerID() int64 { return in.ID }

func (s *server) getPlayerProfile(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in getPlayerProfileInput) (PlayerProfileDTO, error) {
	from, to, err := parseDateRange(in.From, in.To)
	if err != nil {
		return PlayerProfileDTO{}, err
	}

	offset, limit, err := parsePagination(in.Start, in.Rows)
	if err != nil {
		return PlayerProfileDTO{}, err
	}

	profile, err := s.repos.Players.GetPlayerProfile(ctx, in.ID, from, to, offset, limit)
	if err != nil {
		return PlayerProfileDTO{}, notFound(err, "player")
	}

	return fromPlayerProfile(profile, caller), nil
}

// listPlayerTablesInput is the input for the list_player_tables tool.
type listPlayerTablesInput struct {
	ID    int64   `json:"id" jsonschema:"the player's numeric id"`
	From  *string `json:"from,omitempty" jsonschema:"optional start of the date range (RFC3339); when omitted, tables are not filtered by date"`
	To    *string `json:"to,omitempty" jsonschema:"optional end of the date range (RFC3339); when omitted, tables are not filtered by date"`
	Start *int64  `json:"start,omitempty" jsonschema:"pagination offset; defaults to 0 and may not be negative"`
	Rows  *int    `json:"rows,omitempty" jsonschema:"number of rows to return; defaults to 100 and is clamped to [1, 100]"`
}

func (in listPlayerTablesInput) targetPlayerID() int64 { return in.ID }

// listPlayerTablesOutput is the output for the list_player_tables tool.
type listPlayerTablesOutput struct {
	Tables []TableWithBalanceDTO `json:"tables" jsonschema:"the tables the player belongs to"`
}

func (s *server) listPlayerTables(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in listPlayerTablesInput) (listPlayerTablesOutput, error) {
	offset, limit, err := parsePagination(in.Start, in.Rows)
	if err != nil {
		return listPlayerTablesOutput{}, err
	}

	// When no date range is provided, return the unfiltered list of tables.
	if in.From == nil && in.To == nil {
		player, err := s.repos.Players.GetPlayerByID(ctx, in.ID)
		if err != nil {
			return listPlayerTablesOutput{}, notFound(err, "player")
		}

		tables, err := s.repos.Players.GetTables(ctx, player, offset, limit)
		if err != nil {
			return listPlayerTablesOutput{}, err
		}

		return listPlayerTablesOutput{Tables: fromTablesWithBalance(tables)}, nil
	}

	from, to, err := parseDateRange(in.From, in.To)
	if err != nil {
		return listPlayerTablesOutput{}, err
	}

	tables, err := s.repos.Players.GetPlayerTablesFiltered(ctx, in.ID, from, to, offset, limit)
	if err != nil {
		return listPlayerTablesOutput{}, err
	}

	return listPlayerTablesOutput{Tables: fromTablesWithBalance(tables)}, nil
}

// listTablesInput is the input for the list_tables tool.
type listTablesInput struct {
	Start *int64 `json:"start,omitempty" jsonschema:"pagination offset; defaults to 0 and may not be negative"`
	Rows  *int   `json:"rows,omitempty" jsonschema:"number of rows to return; defaults to 100 and is clamped to [1, 100]"`
}

// listTablesOutput is the output for the list_tables tool.
type listTablesOutput struct {
	Tables []TableWithEmailDTO `json:"tables" jsonschema:"the tables"`
}

func (s *server) listTables(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in listTablesInput) (listTablesOutput, error) {
	offset, limit, err := parsePagination(in.Start, in.Rows)
	if err != nil {
		return listTablesOutput{}, err
	}

	// Non-admin callers only see the tables they belong to, without creator emails. This is
	// data-scoping, not access control: the tool is open to any authenticated player, but
	// the result set is narrowed by the caller's membership.
	if !caller.IsSiteAdmin {
		player, err := s.repos.Players.GetPlayerByID(ctx, caller.PlayerID)
		if err != nil {
			return listTablesOutput{}, err
		}

		tables, err := s.repos.Players.GetTables(ctx, player, offset, limit)
		if err != nil {
			return listTablesOutput{}, err
		}

		return listTablesOutput{Tables: fromTablesWithBalanceAsEmail(tables)}, nil
	}

	tables, err := s.repos.Tables.GetActiveTables(ctx, offset, limit)
	if err != nil {
		return listTablesOutput{}, err
	}

	return listTablesOutput{Tables: fromTablesWithEmail(tables, caller)}, nil
}

// getTableInput is the input for the get_table tool.
type getTableInput struct {
	UUID string `json:"uuid" jsonschema:"the table's uuid"`
}

// getTableOutput is the output for the get_table tool.
type getTableOutput struct {
	Table      TableDTO `json:"table" jsonschema:"the table"`
	GamesCount int64    `json:"gamesCount" jsonschema:"the number of games played at the table"`
}

func (s *server) getTable(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in getTableInput) (getTableOutput, error) {
	table, err := s.repos.Tables.GetTableByUUID(ctx, in.UUID)
	if err != nil {
		return getTableOutput{}, notFound(err, "table")
	}

	// Soft-deleted tables are invisible to the MCP surface; report as not found.
	if table.Deleted {
		return getTableOutput{}, errNotFound("table")
	}

	count, err := s.repos.Tables.GetGamesCount(ctx, table)
	if err != nil {
		return getTableOutput{}, err
	}

	return getTableOutput{Table: fromTable(table), GamesCount: count}, nil
}

// getTableRosterInput is the input for the get_table_roster tool.
type getTableRosterInput struct {
	UUID string `json:"uuid" jsonschema:"the table's uuid"`
}

// getTableRosterOutput is the output for the get_table_roster tool.
type getTableRosterOutput struct {
	Table   TableDTO         `json:"table" jsonschema:"the table"`
	Players []PlayerTableDTO `json:"players" jsonschema:"the players at the table"`
}

func (s *server) getTableRoster(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in getTableRosterInput) (getTableRosterOutput, error) {
	table, err := s.repos.Tables.GetTableByUUID(ctx, in.UUID)
	if err != nil {
		return getTableRosterOutput{}, notFound(err, "table")
	}

	// Soft-deleted tables are invisible to the MCP surface; report as not found.
	if table.Deleted {
		return getTableRosterOutput{}, errNotFound("table")
	}

	players, err := s.repos.Tables.GetPlayers(ctx, table)
	if err != nil {
		return getTableRosterOutput{}, err
	}

	return getTableRosterOutput{Table: fromTable(table), Players: fromPlayerTables(players, caller)}, nil
}

// listGameTypesInput is the (empty) input for the list_game_types tool.
type listGameTypesInput struct{}

// listGameTypesOutput is the output for the list_game_types tool.
type listGameTypesOutput struct {
	GameTypes []string `json:"gameTypes" jsonschema:"the registered game-type identifiers"`
}

func (s *server) listGameTypes(_ context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, _ listGameTypesInput) (listGameTypesOutput, error) {
	return listGameTypesOutput{GameTypes: gamefactory.Names()}, nil
}
