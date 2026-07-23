package model

import (
	"context"
	"database/sql"
	"time"
)

// PlayerTransaction is a single row from the players_tables_transactions ledger,
// enriched with the table it belongs to and (when the adjustment came from a
// game) the game's type. Adjustment, PreviousBalance, and CurrentBalance are raw
// integer cents.
type PlayerTransaction struct {
	ID              int64
	Created         time.Time
	Adjustment      int
	PreviousBalance int
	CurrentBalance  int
	Reason          string
	GameID          *int64
	GameType        *string
	TableUUID       string
	TableName       string
}

// GetPlayerTransactions returns a paginated slice of a player's ledger entries,
// newest first. The query crosses into tables and filters NOT t.deleted, so
// transactions at soft-deleted tables are invisible. When tableUUID is non-nil the
// results are narrowed to that single table. The games table is LEFT JOINed so the
// game type is populated for game-driven adjustments and left nil otherwise.
func (r *PlayerRepo) GetPlayerTransactions(ctx context.Context, playerID int64, tableUUID *string, offset int64, limit int) ([]*PlayerTransaction, error) {
	const query = `
SELECT ptt.id, ptt.created, ptt.adjustment, ptt.previous_balance, ptt.current_balance,
       ptt.reason, ptt.game_id, g.game_type, t.uuid, t.name
FROM players_tables_transactions ptt
INNER JOIN players_tables pt ON ptt.players_tables_id = pt.id
INNER JOIN tables t ON pt.table_uuid = t.uuid
LEFT JOIN games g ON ptt.game_id = g.id
WHERE pt.player_id = $1
  AND NOT t.deleted
  AND ($2::uuid IS NULL OR t.uuid = $2::uuid)
ORDER BY ptt.created DESC, ptt.id DESC
OFFSET $3
LIMIT $4`

	rows, err := r.db.QueryContext(ctx, query, playerID, tableUUIDArg(tableUUID), offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*PlayerTransaction, 0)
	for rows.Next() {
		var pt PlayerTransaction
		var reason sql.NullString
		var gameID sql.NullInt64
		var gameType sql.NullString

		if err := rows.Scan(&pt.ID, &pt.Created, &pt.Adjustment, &pt.PreviousBalance, &pt.CurrentBalance,
			&reason, &gameID, &gameType, &pt.TableUUID, &pt.TableName); err != nil {
			return nil, err
		}

		pt.Reason = reason.String
		if gameID.Valid {
			id := gameID.Int64
			pt.GameID = &id
		}
		if gameType.Valid {
			gt := gameType.String
			pt.GameType = &gt
		}

		records = append(records, &pt)
	}

	return records, nil
}

// GetPlayerTransactionsCount returns the total number of ledger entries matching
// GetPlayerTransactions (same NOT t.deleted and optional table filters) for
// pagination totals.
func (r *PlayerRepo) GetPlayerTransactionsCount(ctx context.Context, playerID int64, tableUUID *string) (int64, error) {
	const query = `
SELECT COUNT(*)
FROM players_tables_transactions ptt
INNER JOIN players_tables pt ON ptt.players_tables_id = pt.id
INNER JOIN tables t ON pt.table_uuid = t.uuid
WHERE pt.player_id = $1
  AND NOT t.deleted
  AND ($2::uuid IS NULL OR t.uuid = $2::uuid)`

	var count int64
	if err := r.db.QueryRowContext(ctx, query, playerID, tableUUIDArg(tableUUID)).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// tableUUIDArg turns an optional table uuid into a query argument: the dereferenced
// string when set, or nil (SQL NULL) when not, so the same query serves both the
// filtered and unfiltered cases.
func tableUUIDArg(tableUUID *string) interface{} {
	if tableUUID == nil {
		return nil
	}

	return *tableUUID
}
