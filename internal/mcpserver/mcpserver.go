// Package mcpserver exposes a read-only Model Context Protocol (MCP) server
// over the poker service's data-access repositories. It is served as a
// stateless streamable HTTP handler so that each POST is independent.
package mcpserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mondaynightpoker-server/pkg/model"
)

const (
	defaultRows = 100
	maxRows     = 100
	minRows     = 1
)

// server bundles the repositories that tool handlers close over. registry records the
// access policy declared for each registered tool (populated by registerTool) so an
// exhaustiveness test can verify every tool is classified and none bypasses registration.
type server struct {
	repos    *model.Repositories
	registry map[string]accessPolicy
}

// serverInstructions is sent to clients in the initialize result. Its main job is to
// pin down the unit of every monetary value, since the fields carry raw cents and a
// client that assumes dollars would be off by a factor of one hundred.
const serverInstructions = `Monday Night Poker exposes read-only data about players, tables, and game history.

All monetary values are integers denominated in CENTS, never dollars. A field named
"balanceCents" with the value 150 means one dollar and fifty cents. Never present a
raw cents value to the user as if it were dollars.

Where a field has a "...Display" counterpart (for example balanceDisplay alongside
balanceCents), that string is already formatted as dollars — prefer it verbatim when
showing an amount to the user. For cents-only fields, such as the per-game winnings
map and the profile graph points, divide by 100 and format as dollars before
displaying. Use the raw cents values for any arithmetic or comparison.`

// New builds a stateless streamable HTTP handler exposing the read-only MCP
// tools. Each POST request is handled independently; no long-lived SSE stream
// is used.
func New(repos *model.Repositories, version string) http.Handler {
	s := &server{repos: repos, registry: make(map[string]accessPolicy)}

	mcpServer := mcp.NewServer(
		&mcp.Implementation{Name: "mondaynightpoker", Version: version},
		&mcp.ServerOptions{Instructions: serverInstructions},
	)
	s.registerTools(mcpServer)

	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return mcpServer },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)
}

// parsePagination resolves optional start/rows inputs into an offset and limit.
// start defaults to 0 and may not be negative. rows defaults to 100 and is
// clamped to the range [1, 100].
func parsePagination(start *int64, rows *int) (offset int64, limit int, err error) {
	offset = 0
	if start != nil {
		if *start < 0 {
			return 0, 0, errors.New("start cannot be less than zero")
		}
		offset = *start
	}

	limit = defaultRows
	if rows != nil {
		limit = *rows
		if limit > maxRows {
			limit = maxRows
		}
		if limit < minRows {
			limit = minRows
		}
	}

	return offset, limit, nil
}

// pageTotal resolves a paginated result's total row count and hasMore flag. A
// short page already proves the total (offset + rows returned), so the count
// query is skipped; a full page — or an empty page at a non-zero offset, which
// proves nothing about the total — falls back to the count callback. Every list
// tool computes its pagination metadata here so the arithmetic lives in one place.
func pageTotal(ctx context.Context, offset int64, limit, returned int, count func(context.Context) (int64, error)) (total int64, hasMore bool, err error) {
	if returned < limit && (returned > 0 || offset == 0) {
		return offset + int64(returned), false, nil
	}

	total, err = count(ctx)
	if err != nil {
		return 0, false, err
	}

	return total, offset+int64(returned) < total, nil
}

// parseDateRange resolves optional RFC3339 from/to inputs into a time range.
// When omitted, from defaults to the Unix epoch and to defaults to now (with a
// small margin, mirroring internal/mux). All times are normalized to UTC.
func parseDateRange(from, to *string) (fromTime, toTime time.Time, err error) {
	fromTime = time.Unix(0, 0).UTC()
	toTime = time.Now().In(time.UTC).Add(24 * time.Hour)

	if from != nil {
		parsed, err := time.Parse(time.RFC3339, *from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' date (expected RFC3339): %w", err)
		}
		fromTime = parsed.In(time.UTC)
	}

	if to != nil {
		parsed, err := time.Parse(time.RFC3339, *to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' date (expected RFC3339): %w", err)
		}
		toTime = parsed.In(time.UTC)
	}

	return fromTime, toTime, nil
}

// notFound converts a sql.ErrNoRows into a friendly error, otherwise returns
// the original error unchanged.
func notFound(err error, entity string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return errNotFound(entity)
	}
	return err
}

// errNotFound builds the "not found" error for an entity, matching the message
// notFound produces for a sql.ErrNoRows. Handlers use it to treat a record they
// fetched but must not expose (for example a soft-deleted table) as absent.
func errNotFound(entity string) error {
	return fmt.Errorf("%s not found", entity)
}

// activeTable fetches a table by uuid and reports a soft-deleted one as absent.
// It is the single door through which a tool may turn a uuid into a table:
// TableRepo.GetTableByUUID returns deleted rows (the admin API needs them to
// restore a table), so every MCP caller must reject them, and doing that check
// per handler is what let the aggregates in GetPlayerStats keep summing
// deleted-table money after the table tools had been fixed. Routing every
// lookup through here means a new tool cannot forget the check, the same way
// visibleEmail keeps a new tool from leaking an email.
func (s *server) activeTable(ctx context.Context, uuid string) (*model.Table, error) {
	table, err := s.repos.Tables.GetTableByUUID(ctx, uuid)
	if err != nil {
		return nil, notFound(err, "table")
	}

	if table.Deleted {
		return nil, errNotFound("table")
	}

	return table, nil
}

// gameAtTable resolves a game that belongs to the given table, and is the single
// place that enforces the capability rule for games.
//
// The table uuid is the capability. games.id is sequential, so keying on the id
// alone would let any authenticated caller walk the id space and read every game
// site-wide; a game is only returned when it actually belongs to the table whose
// uuid the caller presented. Every outcome — an unknown or soft-deleted table, an
// unknown game, or a game at some other table — reports "game not found"
// identically, so a caller cannot probe which part of the request was wrong or
// learn that a game with this id exists elsewhere.
//
// withData controls whether the (potentially large) jsonb log is fetched.
// Routing every game lookup through here means a new tool cannot forget the
// ownership comparison, the same way activeTable keeps one from forgetting the
// deleted check.
func (s *server) gameAtTable(ctx context.Context, uuid string, id int64, withData bool) (*model.Game, error) {
	table, err := s.activeTable(ctx, uuid)
	if err != nil {
		return nil, errNotFound("game")
	}

	var game *model.Game
	if withData {
		game, err = s.repos.Games.GetGameByID(ctx, id)
	} else {
		game, err = s.repos.Games.GetGameByIDNoData(ctx, id)
	}
	if err != nil {
		return nil, errNotFound("game")
	}

	if game.TableUUID != table.UUID {
		return nil, errNotFound("game")
	}

	return game, nil
}
