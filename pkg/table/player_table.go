package table

import (
	"context"
	"mondaynightpoker-server/pkg/db"
	"time"
)

const playerTableColumns = `
players_tables.id,
players_tables.player_id,
players_tables.table_uuid,
players_tables.is_table_admin,
players_tables.can_start,
players_tables.can_restart,
players_tables.can_terminate,
players_tables.balance,
players_tables.active,
players_tables.created,
players_tables.updated`

// PlayerTable represents a row in the players_tables table
type PlayerTable struct {
	Player       *Player   `json:"player"`
	PlayerID     int64     `json:"playerId"`
	TableUUID    string    `json:"tableUuid"`
	ID           int64     `json:"id"`
	IsTableAdmin bool      `json:"isTableAdmin"`
	CanStart     bool      `json:"canStart"`
	CanRestart   bool      `json:"canRestart"`
	CanTerminate bool      `json:"canTerminate"`
	Balance      int       `json:"balance"`
	Active       bool      `json:"active"`
	Created      time.Time `json:"created"`
	Updated      time.Time `json:"updated"`
}

func getPlayerTableByRow(row db.Scanner) (*PlayerTable, error) {
	var p Player
	var pt PlayerTable

	if err := row.Scan(&p.ID, &p.Email, &p.DisplayName, &p.IsSiteAdmin, &p.passwordHash, &p.Created, &p.Updated,
		&pt.ID, &pt.PlayerID, &pt.TableUUID, &pt.IsTableAdmin, &pt.CanStart, &pt.CanRestart, &pt.CanTerminate,
		&pt.Balance, &pt.Active, &pt.Created, &pt.Updated); err != nil {
		return nil, err
	}

	pt.Player = &p

	return &pt, nil
}

// AdjustBalance will adjust the balance of the player at the table
func (p *PlayerTable) AdjustBalance(ctx context.Context, byAmount int, reason string, game *Game) error {
	const query = `SELECT adjust_balance($1, $2, $3, $4, $5)`
	var gameID *int64
	if game != nil {
		gameID = &game.ID
	}

	_, err := db.Instance().ExecContext(ctx, query, p.ID, p.Balance, byAmount, gameID, reason)
	if err != nil {
		return err
	}

	p.Balance += byAmount

	return nil
}

// SetActive sets the active state for the player table in the database
func (p *PlayerTable) SetActive(ctx context.Context, active bool) error {
	const query = `
UPDATE players_tables
SET active = $1, updated = (NOW() AT TIME ZONE 'UTC')
WHERE id = $2`
	execContext, err := db.Instance().ExecContext(ctx, query, active, p.ID)
	if err != nil {
		return err
	}

	if ra, _ := execContext.RowsAffected(); ra > 0 {
		p.Active = active
	}

	return nil
}

// Save will save non-balance values
func (p *PlayerTable) Save(ctx context.Context) error {
	const query = `
UPDATE players_tables
SET is_table_admin = $1, can_start = $2, can_restart = $3, can_terminate = $4, updated = (NOW() AT TIME ZONE 'utc')
WHERE id = $5`

	_, err := db.Instance().ExecContext(ctx, query, p.IsTableAdmin, p.CanStart, p.CanRestart, p.CanTerminate, p.ID)
	return err
}
