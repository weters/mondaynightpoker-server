package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"mondaynightpoker-server/pkg/db"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Game is a record in the `games` table
type Game struct {
	ID        int64
	ParentID  int64
	TableUUID string
	GameType  string
	data      interface{}
	rawData   json.RawMessage
	Created   time.Time
	Ended     time.Time
}

const gamesColumns = `id, parent_id, table_uuid, game_type, data, created, ended`

// gamesColumnsNoData is gamesColumns without the jsonb data column. Game logs can
// be large, so list queries omit it and callers fetch a single game by id when the
// log is actually needed.
const gamesColumnsNoData = `id, parent_id, table_uuid, game_type, created, ended`

// Data returns the game's log/state payload (the jsonb `data` column) decoded into
// a generic value. It is nil on games loaded without the data column (e.g.
// ListGamesByTable).
func (g *Game) Data() interface{} {
	return g.data
}

// RawData returns the game's log/state payload as the bytes stored in the column,
// for callers that decode it into a type of their own. Taking the bytes directly
// avoids marshalling Data back to JSON just to decode it again, and it preserves
// values the generic decode flattens (integers become float64 in an interface{}).
// It is nil on games loaded without the data column.
func (g *Game) RawData() json.RawMessage {
	return g.rawData
}

func gameByRow(row db.Scanner) (*Game, error) {
	var parentID sql.NullInt64
	var g Game
	var data []byte
	var ended sql.NullTime

	if err := row.Scan(&g.ID, &parentID, &g.TableUUID, &g.GameType, &data, &g.Created, &ended); err != nil {
		return nil, err
	}

	g.ParentID = parentID.Int64
	if data != nil {
		g.rawData = data
		if err := json.Unmarshal(data, &g.data); err != nil {
			return nil, err
		}
	}

	g.Ended = ended.Time

	return &g, nil
}

// gameByRowNoData scans a game row selected with gamesColumnsNoData (the data
// column omitted). The returned game's data field is left nil.
func gameByRowNoData(row db.Scanner) (*Game, error) {
	var parentID sql.NullInt64
	var g Game
	var ended sql.NullTime

	if err := row.Scan(&g.ID, &parentID, &g.TableUUID, &g.GameType, &g.Created, &ended); err != nil {
		return nil, err
	}

	g.ParentID = parentID.Int64
	g.Ended = ended.Time

	return &g, nil
}

// CreateGame will create a new game for the table
func (r *GameRepo) CreateGame(ctx context.Context, t *Table, gameType string) (*Game, error) {
	const query = `
INSERT INTO games (parent_id, table_uuid, game_type)
VALUES ($1, $2, $3)
RETURNING ` + gamesColumns

	row := r.db.QueryRowContext(ctx, query, nil, t.UUID, gameType)
	return gameByRow(row)
}

// EndGame will end the game and set the data
func (r *GameRepo) EndGame(ctx context.Context, g *Game, data interface{}, balanceAdjustments map[int64]int) error {
	tables := &TableRepo{db: r.db}

	tbl, err := tables.GetTableByUUID(ctx, g.TableUUID)
	if err != nil {
		return err
	}

	players, err := tables.GetPlayers(ctx, tbl)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	commit := false
	defer func() {
		if !commit {
			if err := tx.Rollback(); err != nil {
				logrus.WithError(err).Error("could not rollback transaction")
				return
			}
		}

		if err := tx.Commit(); err != nil {
			logrus.WithError(err).Error("could not commit transaction")
		}
	}()

	g.data = data
	const query = `
UPDATE games
SET data = $1, ended = NOW() AT TIME ZONE 'UTC'
WHERE id = $2
RETURNING ended`

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	row := tx.QueryRowContext(ctx, query, b, g.ID)
	var ended time.Time
	if err := row.Scan(&ended); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, "SELECT adjust_balance($1, $2, $3, $4, $5)")
	if err != nil {
		return err
	}

	for _, player := range players {
		change, found := balanceAdjustments[player.PlayerID]
		if !found {
			logrus.WithField("player", player.PlayerID).Warn("could not find player's balance adjustment")
			continue
		}

		_, err := stmt.ExecContext(ctx, player.ID, player.Balance, change, g.ID, "game ended")
		if err != nil {
			return err
		}
	}

	commit = true
	g.Ended = ended
	return nil
}

// GetGameByID returns a single game by its id, including the jsonb data column.
func (r *GameRepo) GetGameByID(ctx context.Context, id int64) (*Game, error) {
	const query = `
SELECT ` + gamesColumns + `
FROM games
WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	return gameByRow(row)
}

// GetGameByIDNoData returns a single game by its id without the jsonb data column,
// for callers that don't need the (potentially large) game log.
func (r *GameRepo) GetGameByIDNoData(ctx context.Context, id int64) (*Game, error) {
	const query = `
SELECT ` + gamesColumnsNoData + `
FROM games
WHERE id = $1`

	return gameByRowNoData(r.db.QueryRowContext(ctx, query, id))
}

// ListGamesByTable returns a paginated list of games at the table, ordered newest
// first. The jsonb data column is omitted because game logs can be large; callers
// that need a game's log fetch it individually via GetGameByID.
func (r *GameRepo) ListGamesByTable(ctx context.Context, t *Table, offset int64, limit int) ([]*Game, error) {
	const query = `
SELECT ` + gamesColumnsNoData + `
FROM games
WHERE table_uuid = $1
ORDER BY created DESC, id DESC
OFFSET $2
LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, t.UUID, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	games := make([]*Game, 0)
	for rows.Next() {
		g, err := gameByRowNoData(rows)
		if err != nil {
			return nil, err
		}

		games = append(games, g)
	}

	return games, nil
}

// GameAdjustment is a single player's balance adjustment within a game.
type GameAdjustment struct {
	PlayerID    int64
	DisplayName string
	Adjustment  int
}

// GetGameAdjustments returns the per-player balance adjustments for the given game
// ids, keyed by game id. Within each game the adjustments are ordered by amount
// descending (biggest winner first), with a secondary sort on player id for
// deterministic ordering. Games with no ledger rows are absent from the map.
func (r *GameRepo) GetGameAdjustments(ctx context.Context, gameIDs []int64) (map[int64][]*GameAdjustment, error) {
	const query = `
SELECT ptt.game_id, pt.player_id, players.display_name, ptt.adjustment
FROM players_tables_transactions ptt
INNER JOIN players_tables pt ON ptt.players_tables_id = pt.id
INNER JOIN players ON pt.player_id = players.id
WHERE ptt.game_id = ANY($1)
ORDER BY ptt.game_id ASC, ptt.adjustment DESC, pt.player_id ASC`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(gameIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	adjustments := make(map[int64][]*GameAdjustment)
	for rows.Next() {
		var gameID int64
		var ga GameAdjustment
		if err := rows.Scan(&gameID, &ga.PlayerID, &ga.DisplayName, &ga.Adjustment); err != nil {
			return nil, err
		}

		adjustments[gameID] = append(adjustments[gameID], &ga)
	}

	return adjustments, nil
}
