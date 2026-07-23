package model

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"mondaynightpoker-server/pkg/db"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// tableCreationCoolDown is how long non-admins must wait before creating another table
const tableCreationCoolDown = time.Minute

const tableColumns = `
tables.uuid,
tables.name,
tables.player_id,
tables.created,
tables.modified,
tables.deleted`

// Table represents a poker table
// A table has many players and can have many games
type Table struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	// PlayerID is who created the table
	PlayerID int64     `json:"playerId"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Deleted  bool      `json:"deleted"`
}

// TableWithPlayerEmail is a table with the player email who created it
type TableWithPlayerEmail struct {
	*Table
	Email string `json:"playerEmail"`
}

// ErrPlayerNotAtTable happens when user is not a member of the table
var ErrPlayerNotAtTable = errors.New("player is not a member of the table")

// CreateTable creates a new table
func (r *TableRepo) CreateTable(ctx context.Context, p *Player, name string) (*Table, error) {
	if err := r.canCreateTable(ctx, p); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	u := uuid.New().String()
	const query = `
INSERT INTO tables (uuid, name, player_id)
VALUES ($1, $2, $3)
RETURNING created, modified, deleted
`
	var created, modified time.Time
	var deleted bool
	row := tx.QueryRowContext(ctx, query, u, name, p.ID)
	if err := row.Scan(&created, &modified, &deleted); err != nil {
		rollback(tx)
		return nil, err
	}

	const query2 = `
INSERT INTO players_tables (player_id, table_uuid, is_table_admin)
VALUES ($1, $2, true)`
	if _, err = tx.ExecContext(ctx, query2, p.ID, u); err != nil {
		rollback(tx)
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Table{
		UUID:     u,
		Name:     name,
		PlayerID: p.ID,
		Created:  created,
		Modified: modified,
		Deleted:  deleted,
	}, nil
}

// CloneTable creates a new table from an existing one. Authorization is the
// caller's responsibility. All players from the source are added to the new
// table in randomized order with their balances zeroed and active set to false
// (sit-out). Table stake and admin/permission flags are carried over.
func (r *TableRepo) CloneTable(ctx context.Context, p *Player, source *Table, name string) (*Table, error) {
	if err := r.canCreateTable(ctx, p); err != nil {
		return nil, err
	}

	sourcePlayers, err := r.GetPlayers(ctx, source)
	if err != nil {
		return nil, err
	}

	rand.Shuffle(len(sourcePlayers), func(i, j int) {
		sourcePlayers[i], sourcePlayers[j] = sourcePlayers[j], sourcePlayers[i]
	})

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	u := uuid.New().String()
	const tableQuery = `
INSERT INTO tables (uuid, name, player_id)
VALUES ($1, $2, $3)
RETURNING created, modified, deleted`

	var created, modified time.Time
	var deleted bool
	if err := tx.QueryRowContext(ctx, tableQuery, u, name, p.ID).Scan(&created, &modified, &deleted); err != nil {
		rollback(tx)
		return nil, err
	}

	const ptQuery = `
INSERT INTO players_tables (player_id, table_uuid, is_table_admin, can_start, can_restart, can_terminate, table_stake, active, is_blocked)
VALUES ($1, $2, $3, $4, $5, $6, $7, false, $8)`

	for _, src := range sourcePlayers {
		if _, err := tx.ExecContext(ctx, ptQuery,
			src.PlayerID, u,
			src.IsTableAdmin, src.CanStart, src.CanRestart, src.CanTerminate,
			src.TableStake, src.IsBlocked,
		); err != nil {
			rollback(tx)
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Table{
		UUID:     u,
		Name:     name,
		PlayerID: p.ID,
		Created:  created,
		Modified: modified,
		Deleted:  deleted,
	}, nil
}

// canCreateTable will see if the user is allowed to create a table
// returns nil if the user can create a table
func (r *TableRepo) canCreateTable(ctx context.Context, p *Player) error {
	// site admins can always create a table
	if p.IsSiteAdmin {
		return nil
	}

	const query = `
SELECT COUNT(*)
FROM tables
WHERE player_id = $1
  AND created >= $2 AT TIME ZONE 'UTC'`

	row := r.db.QueryRowContext(ctx, query, p.ID, time.Now().In(time.UTC).Add(tableCreationCoolDown*-1))
	var count int
	if err := row.Scan(&count); err != nil {
		return err
	}

	if count > 0 {
		return UserError("you must wait before you create another table")
	}

	return nil
}

func getTableByRow(row db.Scanner, additionalColumns ...interface{}) (*Table, error) {
	var t Table
	columns := []interface{}{
		&t.UUID,
		&t.Name,
		&t.PlayerID,
		&t.Created,
		&t.Modified,
		&t.Deleted,
	}

	if len(additionalColumns) > 0 {
		columns = append(columns, additionalColumns...)
	}

	if err := row.Scan(columns...); err != nil {
		return nil, err
	}

	return &t, nil
}

// GetTableByUUID returns a table by its UUID
func (r *TableRepo) GetTableByUUID(ctx context.Context, uuid string) (*Table, error) {
	const query = `
SELECT ` + tableColumns + `
FROM tables
WHERE uuid = $1`

	row := r.db.QueryRowContext(ctx, query, uuid)
	return getTableByRow(row)
}

// GetTables returns a paginated list of tables, including soft-deleted ones.
func (r *TableRepo) GetTables(ctx context.Context, offset int64, limit int) ([]*TableWithPlayerEmail, error) {
	const query = `
SELECT ` + tableColumns + `, players.email
FROM tables
INNER JOIN players ON tables.player_id = players.id
ORDER BY tables.created DESC, tables.uuid DESC
OFFSET $1
LIMIT $2`

	return scanTablesWithPlayerEmail(r.db.QueryContext(ctx, query, offset, limit))
}

// GetActiveTables returns a paginated list of non-deleted tables. The deleted
// filter is applied in SQL so pagination stays consistent (offset/limit are not
// distorted by a post-fetch filter).
func (r *TableRepo) GetActiveTables(ctx context.Context, offset int64, limit int) ([]*TableWithPlayerEmail, error) {
	const query = `
SELECT ` + tableColumns + `, players.email
FROM tables
INNER JOIN players ON tables.player_id = players.id
WHERE NOT tables.deleted
ORDER BY tables.created DESC, tables.uuid DESC
OFFSET $1
LIMIT $2`

	return scanTablesWithPlayerEmail(r.db.QueryContext(ctx, query, offset, limit))
}

// scanTablesWithPlayerEmail collects the rows from a tables-with-email query
// (the tableColumns plus players.email) into TableWithPlayerEmail records. It
// takes the raw QueryContext result so the deleted/active variants differ only
// in their SQL.
func scanTablesWithPlayerEmail(rows *sql.Rows, err error) ([]*TableWithPlayerEmail, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]*TableWithPlayerEmail, 0)
	for rows.Next() {
		var email string
		row, err := getTableByRow(rows, &email)
		if err != nil {
			return nil, err
		}

		tables = append(tables, &TableWithPlayerEmail{
			Table: row,
			Email: email,
		})
	}

	return tables, nil
}

// GetActiveTablesFiltered returns a paginated list of non-deleted tables created
// within the given range. It is GetActiveTables with a date filter on
// tables.created (a table is one night by convention). The deleted filter stays in
// SQL so pagination is not distorted by a post-fetch filter.
func (r *TableRepo) GetActiveTablesFiltered(ctx context.Context, from, to time.Time, offset int64, limit int) ([]*TableWithPlayerEmail, error) {
	const query = `
SELECT ` + tableColumns + `, players.email
FROM tables
INNER JOIN players ON tables.player_id = players.id
WHERE NOT tables.deleted
  AND tables.created >= $1 AND tables.created <= $2
ORDER BY tables.created DESC, tables.uuid DESC
OFFSET $3
LIMIT $4`

	return scanTablesWithPlayerEmail(r.db.QueryContext(ctx, query, from, to, offset, limit))
}

// GetActiveTablesCount returns the total number of non-deleted tables created
// within the given range, for pagination totals.
func (r *TableRepo) GetActiveTablesCount(ctx context.Context, from, to time.Time) (int64, error) {
	const query = `
SELECT COUNT(*)
FROM tables
WHERE NOT tables.deleted
  AND tables.created >= $1 AND tables.created <= $2`

	var count int64
	if err := r.db.QueryRowContext(ctx, query, from, to).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// TablePlayerStats is a single roster member's summary at a table: their current
// balance plus per-game aggregates. Balance and NetWinnings are raw integer cents.
type TablePlayerStats struct {
	PlayerID    int64
	DisplayName string
	Balance     int
	GamesPlayed int
	NetWinnings int
}

// GetTableStats returns one TablePlayerStats per roster member at the table.
// Balance is the current players_tables balance. GamesPlayed and NetWinnings are
// computed from game-driven ledger rows only (game_id IS NOT NULL); the ledger is
// LEFT JOINed so members who have not played a game still appear with zeros.
// Results are ordered by net winnings descending, then player id ascending.
func (r *TableRepo) GetTableStats(ctx context.Context, t *Table) ([]*TablePlayerStats, error) {
	const query = `
SELECT pt.player_id,
       players.display_name,
       pt.balance,
       COUNT(DISTINCT ptt.game_id) FILTER (WHERE ptt.game_id IS NOT NULL) AS games_played,
       COALESCE(SUM(ptt.adjustment) FILTER (WHERE ptt.game_id IS NOT NULL), 0) AS net_winnings
FROM players_tables pt
INNER JOIN players ON pt.player_id = players.id
LEFT JOIN players_tables_transactions ptt ON ptt.players_tables_id = pt.id
WHERE pt.table_uuid = $1
GROUP BY pt.player_id, players.display_name, pt.balance
ORDER BY net_winnings DESC, pt.player_id ASC`

	rows, err := r.db.QueryContext(ctx, query, t.UUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]*TablePlayerStats, 0)
	for rows.Next() {
		var s TablePlayerStats
		if err := rows.Scan(&s.PlayerID, &s.DisplayName, &s.Balance, &s.GamesPlayed, &s.NetWinnings); err != nil {
			return nil, err
		}

		stats = append(stats, &s)
	}

	return stats, nil
}

// TableAggregates summarizes a table: how many players are seated, how many games
// have been played, and the sum of every seated player's balance (raw cents).
type TableAggregates struct {
	PlayersCount int
	GamesCount   int64
	TotalBalance int
}

// GetTableAggregates returns the roster size, games count, and total balance for
// the table in a single query.
func (r *TableRepo) GetTableAggregates(ctx context.Context, t *Table) (*TableAggregates, error) {
	const query = `
SELECT
  (SELECT COUNT(*) FROM players_tables WHERE table_uuid = $1),
  (SELECT COUNT(*) FROM games WHERE table_uuid = $1),
  (SELECT COALESCE(SUM(balance), 0) FROM players_tables WHERE table_uuid = $1)`

	var agg TableAggregates
	if err := r.db.QueryRowContext(ctx, query, t.UUID).Scan(&agg.PlayersCount, &agg.GamesCount, &agg.TotalBalance); err != nil {
		return nil, err
	}

	return &agg, nil
}

// GetActivePlayersShifted returns all the active players at the table with the players shifted by the number of games
func (r *TableRepo) GetActivePlayersShifted(ctx context.Context, t *Table) ([]*PlayerTable, error) {
	players, err := r.GetPlayers(ctx, t)
	if err != nil {
		return nil, err
	}

	activePlayers := make([]*PlayerTable, 0, len(players))
	for _, player := range players {
		if player.Active {
			activePlayers = append(activePlayers, player)
		}
	}

	if len(activePlayers) == 0 {
		return []*PlayerTable{}, nil
	}

	count, err := r.GetGamesCount(ctx, t)
	if err != nil {
		return nil, err
	}

	offset := int(count % int64(len(activePlayers)))
	if offset == 0 {
		return players, nil
	}

	tail := activePlayers[offset:]
	head := activePlayers[:offset]
	return append(tail, head...), nil
}

// GetPlayers returns all players at the table
func (r *TableRepo) GetPlayers(ctx context.Context, t *Table) ([]*PlayerTable, error) {
	const query = `
SELECT ` + playerColumns + `, ` + playerTableColumns + `
FROM players_tables
INNER JOIN players ON players_tables.player_id = players.id
WHERE players_tables.table_uuid = $1
ORDER BY players_tables.id`

	rows, err := r.db.QueryContext(ctx, query, t.UUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*PlayerTable, 0)
	for rows.Next() {
		p, err := getPlayerTableByRow(rows)
		if err != nil {
			return nil, err
		}

		records = append(records, p)
	}

	return records, nil
}

// GetGamesCount returns the number of games played by the table
func (r *TableRepo) GetGamesCount(ctx context.Context, t *Table) (int64, error) {
	const query = `
SELECT COUNT(id)
FROM games
WHERE table_uuid = $1`

	var count int64
	if err := r.db.QueryRowContext(ctx, query, t.UUID).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// Save saves any changes
func (r *TableRepo) Save(ctx context.Context, t *Table) error {
	const query = `
UPDATE tables
SET name = $1,
    deleted = $2,
    modified = (NOW() AT TIME ZONE 'UTC')
WHERE uuid = $3`

	_, err := r.db.ExecContext(ctx, query, t.Name, t.Deleted, t.UUID)
	return err
}

func rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil {
		logrus.WithError(err).Error("could not rollback transaction")
	}
}
