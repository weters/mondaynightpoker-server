package model

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
players_tables.table_stake,
players_tables.active,
players_tables.is_blocked,
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
	TableStake   int       `json:"tableStake"`
	Active       bool      `json:"active"`
	IsBlocked    bool      `json:"isBlocked"`
	Created      time.Time `json:"created"`
	Updated      time.Time `json:"updated"`
}

func getPlayerTableByRow(row db.Scanner) (*PlayerTable, error) {
	var p Player
	var pt PlayerTable

	if err := row.Scan(&p.ID, &p.Email, &p.DisplayName, &p.IsSiteAdmin, &p.Status, &p.passwordHash, &p.Created, &p.Updated,
		&pt.ID, &pt.PlayerID, &pt.TableUUID, &pt.IsTableAdmin, &pt.CanStart, &pt.CanRestart, &pt.CanTerminate,
		&pt.Balance, &pt.TableStake, &pt.Active, &pt.IsBlocked, &pt.Created, &pt.Updated); err != nil {
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

// Save will save non-balance values
func (p *PlayerTable) Save(ctx context.Context) error {
	const query = `
UPDATE players_tables
SET active = $1,
    table_stake = $2,
    is_table_admin = $3,
    can_start = $4,
    can_restart = $5,
    can_terminate = $6,
    is_blocked = $7,
    updated = (NOW() AT TIME ZONE 'utc')
WHERE id = $8`

	_, err := db.Instance().ExecContext(ctx, query, p.Active, p.TableStake, p.IsTableAdmin, p.CanStart, p.CanRestart, p.CanTerminate, p.IsBlocked, p.ID)
	return err
}

// IsPlaying returns true if the player should be dealt in the next hand
// This will return false if player is marked as not active, or they are blocked (by table admin)
func (p *PlayerTable) IsPlaying() bool {
	return !p.IsBlocked && p.Active
}

// GetPlayerID returns the player ID
func (p *PlayerTable) GetPlayerID() int64 {
	return p.PlayerID
}

// GetTableStake returns the table stake
// This method returns the player's balance, unless their balance is below their table stake. In that case,
// it returns the table stake.
func (p *PlayerTable) GetTableStake() int {
	if p.Balance > p.TableStake {
		return p.Balance
	}

	return p.TableStake
}
