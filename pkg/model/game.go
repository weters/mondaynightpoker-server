package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
)

// Game is a record in the `games` table
type Game struct {
	ID        int64
	ParentID  int64
	TableUUID string
	GameType  string
	data      interface{}
	Created   time.Time
	Ended     time.Time
}

const gamesColumns = `id, parent_id, table_uuid, game_type, data, created, ended`

func gameByRow(row *sql.Row) (*Game, error) {
	var parentID sql.NullInt64
	var g Game
	var data []byte
	var ended sql.NullTime

	if err := row.Scan(&g.ID, &parentID, &g.TableUUID, &g.GameType, &data, &g.Created, &ended); err != nil {
		return nil, err
	}

	g.ParentID = parentID.Int64
	if data != nil {
		if err := json.Unmarshal(data, &g.data); err != nil {
			return nil, err
		}
	}

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
