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
