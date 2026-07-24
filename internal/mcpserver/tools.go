package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mondaynightpoker-server/internal/oauth"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/money"
	"mondaynightpoker-server/pkg/room/gamefactory"
)

// registerTools registers all read-only tools on the given MCP server. Every tool is
// installed through registerTool with a mandatory access policy; the handlers below contain
// no authorization checks of their own and receive the already-authorized caller.
func (s *server) registerTools(m *mcp.Server) {
	registerTool(s, m, &mcp.Tool{
		Name:        "whoami",
		Description: "Identify the calling player: returns the player record for whoever the access token belongs to, including their email. Takes no arguments. Use this to resolve \"me\" or \"my\" into the numeric player id the other tools require.",
	}, accessAuthenticated, s.whoami)

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
		Description: "Get a single table by its uuid, including aggregate counts (games played, players seated, and the total of every seated player's balance). Available to any authenticated player (capability-URL semantics).",
	}, accessAuthenticated, s.getTable)

	registerTool(s, m, &mcp.Tool{
		Name:        "get_table_roster",
		Description: "Get the roster of players at a table, including their table permissions and balances (in cents, with preformatted dollar strings alongside). Available to any authenticated player; for non-admin callers every roster member's email is omitted except the caller's own.",
	}, accessAuthenticated, s.getTableRoster)

	registerTool(s, m, &mcp.Tool{
		Name:        "list_table_games",
		Description: "List the games played at a table by its uuid, newest first, with each game's per-player balance adjustments. Available to any authenticated player (capability-URL semantics).",
	}, accessAuthenticated, s.listTableGames)

	registerTool(s, m, &mcp.Tool{
		Name:        "get_game",
		Description: "Get a single game by its table uuid and numeric game id, including its per-player balance adjustments and, when includeLog is true, the full game log. The table uuid is the capability: a game is only returned when it belongs to that table. Available to any authenticated player (capability-URL semantics).",
	}, accessAuthenticated, s.getGame)

	registerTool(s, m, &mcp.Tool{
		Name:        "get_table_stats",
		Description: "Get per-player statistics for a table by its uuid: each roster member's current balance, games played, and net winnings. Available to any authenticated player.",
	}, accessAuthenticated, s.getTableStats)

	registerTool(s, m, &mcp.Tool{
		Name:        "list_player_transactions",
		Description: "List a player's ledger transactions, newest first, optionally narrowed to a single table by uuid. Non-admin callers may only request their own player id.",
	}, accessSelfScoped, s.listPlayerTransactions)

	registerTool(s, m, &mcp.Tool{
		Name:        "leaderboard",
		Description: "Get a leaderboard, scoped to the tables the calling player belongs to (optionally within a date range on table creation), sorted by net winnings descending. Takes no player id: it is always computed for the caller's own tables. Available to any authenticated player.",
	}, accessAuthenticated, s.leaderboard)

	registerTool(s, m, &mcp.Tool{
		Name:        "list_game_types",
		Description: "List the registered game types, each with its canonical display group.",
	}, accessAuthenticated, s.listGameTypes)
}

// whoamiInput is the (empty) input for the whoami tool. It deliberately takes no player
// id: the subject is always the caller carried by the access token, so there is nothing
// for a client to get wrong and nothing to authorize beyond authentication.
type whoamiInput struct{}

func (s *server) whoami(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, _ whoamiInput) (PlayerDTO, error) {
	// Load the record live rather than reporting the token's claims back: a player whose
	// admin bit or display name changed since the token was issued should see the current
	// values, and a player deleted out from under a live token is reported as absent.
	player, err := s.repos.Players.GetPlayerByID(ctx, caller.PlayerID)
	if err != nil {
		return PlayerDTO{}, notFound(err, "player")
	}

	// fromPlayer surfaces the email because the record's owner is the caller.
	return fromPlayer(player, caller), nil
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
	Total   int64       `json:"total" jsonschema:"the total number of players matching the request, ignoring pagination"`
	HasMore bool        `json:"hasMore" jsonschema:"whether more players exist beyond this page"`
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

	total, hasMore, err := pageTotal(ctx, offset, limit, len(players), func(ctx context.Context) (int64, error) {
		return s.repos.Players.GetPlayersCount(ctx, search)
	})
	if err != nil {
		return listPlayersOutput{}, err
	}

	return listPlayersOutput{
		Players: fromPlayers(players, caller),
		Total:   total,
		HasMore: hasMore,
	}, nil
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
	Tables  []TableWithBalanceDTO `json:"tables" jsonschema:"the tables the player belongs to"`
	Total   int64                 `json:"total" jsonschema:"the total number of tables matching the request, ignoring pagination"`
	HasMore bool                  `json:"hasMore" jsonschema:"whether more tables exist beyond this page"`
}

func (s *server) listPlayerTables(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in listPlayerTablesInput) (listPlayerTablesOutput, error) {
	offset, limit, err := parsePagination(in.Start, in.Rows)
	if err != nil {
		return listPlayerTablesOutput{}, err
	}

	// parseDateRange defaults to an epoch..now range, so the filtered query with the
	// default bounds returns exactly the unfiltered list; there is no separate
	// undated code path to keep in sync.
	from, to, err := parseDateRange(in.From, in.To)
	if err != nil {
		return listPlayerTablesOutput{}, err
	}

	// Look the player up first so an unknown id is "player not found" rather than an
	// indistinguishable empty list.
	if _, err := s.repos.Players.GetPlayerByID(ctx, in.ID); err != nil {
		return listPlayerTablesOutput{}, notFound(err, "player")
	}

	tables, err := s.repos.Players.GetPlayerTablesFiltered(ctx, in.ID, from, to, offset, limit)
	if err != nil {
		return listPlayerTablesOutput{}, err
	}

	total, hasMore, err := pageTotal(ctx, offset, limit, len(tables), func(ctx context.Context) (int64, error) {
		return s.repos.Players.GetPlayerTablesFilteredCount(ctx, in.ID, from, to)
	})
	if err != nil {
		return listPlayerTablesOutput{}, err
	}

	return listPlayerTablesOutput{
		Tables:  fromTablesWithBalance(tables),
		Total:   total,
		HasMore: hasMore,
	}, nil
}

// listTablesInput is the input for the list_tables tool.
type listTablesInput struct {
	From  *string `json:"from,omitempty" jsonschema:"optional start of the date range (RFC3339), filtering on table creation; when omitted, tables are not filtered by date"`
	To    *string `json:"to,omitempty" jsonschema:"optional end of the date range (RFC3339), filtering on table creation; when omitted, tables are not filtered by date"`
	Start *int64  `json:"start,omitempty" jsonschema:"pagination offset; defaults to 0 and may not be negative"`
	Rows  *int    `json:"rows,omitempty" jsonschema:"number of rows to return; defaults to 100 and is clamped to [1, 100]"`
}

// listTablesOutput is the output for the list_tables tool.
type listTablesOutput struct {
	Tables  []TableWithEmailDTO `json:"tables" jsonschema:"the tables"`
	Total   int64               `json:"total" jsonschema:"the total number of tables matching the request, ignoring pagination"`
	HasMore bool                `json:"hasMore" jsonschema:"whether more tables exist beyond this page"`
}

func (s *server) listTables(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in listTablesInput) (listTablesOutput, error) {
	offset, limit, err := parsePagination(in.Start, in.Rows)
	if err != nil {
		return listTablesOutput{}, err
	}

	// parseDateRange defaults to an epoch..now range, so the filtered queries with the
	// default bounds return exactly the unfiltered lists; there is no separate undated
	// code path to keep in sync.
	from, to, err := parseDateRange(in.From, in.To)
	if err != nil {
		return listTablesOutput{}, err
	}

	// Non-admin callers only see the tables they belong to, without creator emails. This is
	// data-scoping, not access control: the tool is open to any authenticated player, but
	// the result set is narrowed by the caller's membership.
	if !caller.IsSiteAdmin {
		tables, err := s.repos.Players.GetPlayerTablesFiltered(ctx, caller.PlayerID, from, to, offset, limit)
		if err != nil {
			return listTablesOutput{}, err
		}

		total, hasMore, err := pageTotal(ctx, offset, limit, len(tables), func(ctx context.Context) (int64, error) {
			return s.repos.Players.GetPlayerTablesFilteredCount(ctx, caller.PlayerID, from, to)
		})
		if err != nil {
			return listTablesOutput{}, err
		}

		return listTablesOutput{
			Tables:  fromTablesWithBalanceAsEmail(tables),
			Total:   total,
			HasMore: hasMore,
		}, nil
	}

	tables, err := s.repos.Tables.GetActiveTablesFiltered(ctx, from, to, offset, limit)
	if err != nil {
		return listTablesOutput{}, err
	}

	total, hasMore, err := pageTotal(ctx, offset, limit, len(tables), func(ctx context.Context) (int64, error) {
		return s.repos.Tables.GetActiveTablesCount(ctx, from, to)
	})
	if err != nil {
		return listTablesOutput{}, err
	}

	return listTablesOutput{
		Tables:  fromTablesWithEmail(tables, caller),
		Total:   total,
		HasMore: hasMore,
	}, nil
}

// getTableInput is the input for the get_table tool.
type getTableInput struct {
	UUID string `json:"uuid" jsonschema:"the table's uuid"`
}

// getTableOutput is the output for the get_table tool.
type getTableOutput struct {
	Table               TableDTO `json:"table" jsonschema:"the table"`
	GamesCount          int64    `json:"gamesCount" jsonschema:"the number of games played at the table"`
	PlayersCount        int      `json:"playersCount" jsonschema:"the number of players seated at the table"`
	TotalBalanceCents   int      `json:"totalBalanceCents" jsonschema:"the sum of every seated player's balance, in cents"`
	TotalBalanceDisplay string   `json:"totalBalanceDisplay" jsonschema:"the sum of every seated player's balance, preformatted for display in dollars; show this to the user rather than converting totalBalanceCents yourself"`
}

func (s *server) getTable(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in getTableInput) (getTableOutput, error) {
	table, err := s.activeTable(ctx, in.UUID)
	if err != nil {
		return getTableOutput{}, err
	}

	aggregates, err := s.repos.Tables.GetTableAggregates(ctx, table)
	if err != nil {
		return getTableOutput{}, err
	}

	return getTableOutput{
		Table:               fromTable(table),
		GamesCount:          aggregates.GamesCount,
		PlayersCount:        aggregates.PlayersCount,
		TotalBalanceCents:   aggregates.TotalBalance,
		TotalBalanceDisplay: money.FormatCents(aggregates.TotalBalance),
	}, nil
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
	table, err := s.activeTable(ctx, in.UUID)
	if err != nil {
		return getTableRosterOutput{}, err
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
	GameTypes []GameTypeDTO `json:"gameTypes" jsonschema:"the registered game types, each with its canonical display group"`
}

func (s *server) listGameTypes(_ context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, _ listGameTypesInput) (listGameTypesOutput, error) {
	names := gamefactory.Names()
	types := make([]GameTypeDTO, 0, len(names))
	for _, name := range names {
		factory, err := gamefactory.Get(name)
		if err != nil {
			return listGameTypesOutput{}, err
		}

		types = append(types, GameTypeDTO{ID: name, DisplayGroup: model.GameTypeGroup(factory.DisplayName())})
	}

	return listGameTypesOutput{GameTypes: types}, nil
}

// listTableGamesInput is the input for the list_table_games tool.
type listTableGamesInput struct {
	UUID  string `json:"uuid" jsonschema:"the table's uuid"`
	Start *int64 `json:"start,omitempty" jsonschema:"pagination offset; defaults to 0 and may not be negative"`
	Rows  *int   `json:"rows,omitempty" jsonschema:"number of rows to return; defaults to 100 and is clamped to [1, 100]"`
}

// listTableGamesOutput is the output for the list_table_games tool.
type listTableGamesOutput struct {
	Games   []GameSummaryDTO `json:"games" jsonschema:"the games played at the table, newest first"`
	Total   int64            `json:"total" jsonschema:"the total number of games at the table, ignoring pagination"`
	HasMore bool             `json:"hasMore" jsonschema:"whether more games exist beyond this page"`
}

func (s *server) listTableGames(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in listTableGamesInput) (listTableGamesOutput, error) {
	offset, limit, err := parsePagination(in.Start, in.Rows)
	if err != nil {
		return listTableGamesOutput{}, err
	}

	table, err := s.activeTable(ctx, in.UUID)
	if err != nil {
		return listTableGamesOutput{}, err
	}

	games, err := s.repos.Games.ListGamesByTable(ctx, table, offset, limit)
	if err != nil {
		return listTableGamesOutput{}, err
	}

	total, hasMore, err := pageTotal(ctx, offset, limit, len(games), func(ctx context.Context) (int64, error) {
		return s.repos.Tables.GetGamesCount(ctx, table)
	})
	if err != nil {
		return listTableGamesOutput{}, err
	}

	gameIDs := make([]int64, 0, len(games))
	for _, g := range games {
		gameIDs = append(gameIDs, g.ID)
	}

	adjustments, err := s.repos.Games.GetGameAdjustments(ctx, gameIDs)
	if err != nil {
		return listTableGamesOutput{}, err
	}

	summaries := make([]GameSummaryDTO, 0, len(games))
	for _, g := range games {
		summaries = append(summaries, fromGameSummary(g, adjustments[g.ID]))
	}

	return listTableGamesOutput{
		Games:   summaries,
		Total:   total,
		HasMore: hasMore,
	}, nil
}

// getGameInput is the input for the get_game tool. Both the table uuid and the game
// id are required: the unguessable uuid is the capability, so a game is only returned
// when it belongs to that table. Keying on the sequential game id alone would let any
// authenticated caller enumerate every game site-wide.
type getGameInput struct {
	UUID       string `json:"uuid" jsonschema:"the uuid of the table the game belongs to"`
	ID         int64  `json:"id" jsonschema:"the game's numeric id"`
	IncludeLog *bool  `json:"includeLog,omitempty" jsonschema:"whether to include the full game log; defaults to false because the log can be large"`
}

func (s *server) getGame(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in getGameInput) (GameDTO, error) {
	// Resolve the table first: this both rejects soft-deleted/absent tables and pins the
	// caller to the capability they hold. A caller who was never given this uuid cannot
	// reach the game regardless of how they guessed its (sequential) id.
	table, err := s.activeTable(ctx, in.UUID)
	if err != nil {
		return GameDTO{}, errNotFound("game")
	}

	// Only fetch the (potentially large) jsonb log when the caller asked for it.
	includeLog := in.IncludeLog != nil && *in.IncludeLog

	var game *model.Game
	if includeLog {
		game, err = s.repos.Games.GetGameByID(ctx, in.ID)
	} else {
		game, err = s.repos.Games.GetGameByIDNoData(ctx, in.ID)
	}
	if err != nil {
		// A missing game and a game belonging to another table are reported identically,
		// so a caller cannot probe which part of the request was wrong (or learn that a
		// game with this id exists at some other table).
		return GameDTO{}, errNotFound("game")
	}

	if game.TableUUID != table.UUID {
		return GameDTO{}, errNotFound("game")
	}

	adjustments, err := s.repos.Games.GetGameAdjustments(ctx, []int64{game.ID})
	if err != nil {
		return GameDTO{}, err
	}

	dto := GameDTO{GameSummaryDTO: fromGameSummary(game, adjustments[game.ID])}
	if includeLog {
		dto.Log = game.Data()
	}

	return dto, nil
}

// getTableStatsInput is the input for the get_table_stats tool.
type getTableStatsInput struct {
	UUID string `json:"uuid" jsonschema:"the table's uuid"`
}

// getTableStatsOutput is the output for the get_table_stats tool.
type getTableStatsOutput struct {
	Table   TableDTO              `json:"table" jsonschema:"the table"`
	Players []TablePlayerStatsDTO `json:"players" jsonschema:"per-player statistics for the table, sorted by net winnings descending"`
}

func (s *server) getTableStats(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in getTableStatsInput) (getTableStatsOutput, error) {
	table, err := s.activeTable(ctx, in.UUID)
	if err != nil {
		return getTableStatsOutput{}, err
	}

	stats, err := s.repos.Tables.GetTableStats(ctx, table)
	if err != nil {
		return getTableStatsOutput{}, err
	}

	return getTableStatsOutput{
		Table:   fromTable(table),
		Players: fromTablePlayerStatsList(stats),
	}, nil
}

// listPlayerTransactionsInput is the input for the list_player_transactions tool.
type listPlayerTransactionsInput struct {
	ID        int64   `json:"id" jsonschema:"the player's numeric id"`
	TableUUID *string `json:"tableUuid,omitempty" jsonschema:"optional table uuid; when provided, only transactions at that table are returned"`
	Start     *int64  `json:"start,omitempty" jsonschema:"pagination offset; defaults to 0 and may not be negative"`
	Rows      *int    `json:"rows,omitempty" jsonschema:"number of rows to return; defaults to 100 and is clamped to [1, 100]"`
}

func (in listPlayerTransactionsInput) targetPlayerID() int64 { return in.ID }

// listPlayerTransactionsOutput is the output for the list_player_transactions tool.
type listPlayerTransactionsOutput struct {
	Transactions []TransactionDTO `json:"transactions" jsonschema:"the player's ledger transactions, newest first"`
	Total        int64            `json:"total" jsonschema:"the total number of transactions matching the request, ignoring pagination"`
	HasMore      bool             `json:"hasMore" jsonschema:"whether more transactions exist beyond this page"`
}

func (s *server) listPlayerTransactions(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in listPlayerTransactionsInput) (listPlayerTransactionsOutput, error) {
	offset, limit, err := parsePagination(in.Start, in.Rows)
	if err != nil {
		return listPlayerTransactionsOutput{}, err
	}

	// When a table is named, resolve it through activeTable first so a soft-deleted (or
	// absent) table is a not-found error rather than silently returning no rows.
	if in.TableUUID != nil {
		if _, err := s.activeTable(ctx, *in.TableUUID); err != nil {
			return listPlayerTransactionsOutput{}, err
		}
	}

	transactions, err := s.repos.Players.GetPlayerTransactions(ctx, in.ID, in.TableUUID, offset, limit)
	if err != nil {
		return listPlayerTransactionsOutput{}, err
	}

	total, hasMore, err := pageTotal(ctx, offset, limit, len(transactions), func(ctx context.Context) (int64, error) {
		return s.repos.Players.GetPlayerTransactionsCount(ctx, in.ID, in.TableUUID)
	})
	if err != nil {
		return listPlayerTransactionsOutput{}, err
	}

	return listPlayerTransactionsOutput{
		Transactions: fromTransactions(transactions),
		Total:        total,
		HasMore:      hasMore,
	}, nil
}

// leaderboardInput is the input for the leaderboard tool. It intentionally takes no
// player id: the leaderboard is always scoped to the caller's own tables.
type leaderboardInput struct {
	From *string `json:"from,omitempty" jsonschema:"optional start of the date range (RFC3339), filtering on table creation; defaults to the epoch"`
	To   *string `json:"to,omitempty" jsonschema:"optional end of the date range (RFC3339), filtering on table creation; defaults to now"`
}

// leaderboardOutput is the output for the leaderboard tool.
type leaderboardOutput struct {
	Entries []LeaderboardEntryDTO `json:"entries" jsonschema:"the leaderboard entries, sorted by net winnings descending"`
}

func (s *server) leaderboard(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in leaderboardInput) (leaderboardOutput, error) {
	from, to, err := parseDateRange(in.From, in.To)
	if err != nil {
		return leaderboardOutput{}, err
	}

	entries, err := s.repos.Players.GetLeaderboard(ctx, caller.PlayerID, from, to)
	if err != nil {
		return leaderboardOutput{}, err
	}

	return leaderboardOutput{Entries: fromLeaderboardEntries(entries)}, nil
}
