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

// SavePlayerTable will save non-balance values
func (r *TableRepo) SavePlayerTable(ctx context.Context, pt *PlayerTable) error {
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

	_, err := r.db.ExecContext(ctx, query, pt.Active, pt.TableStake, pt.IsTableAdmin, pt.CanStart, pt.CanRestart, pt.CanTerminate, pt.IsBlocked, pt.ID)
	return err
}

// Save will save non-balance values
//
// Deprecated: use Repositories.Tables.SavePlayerTable instead.
func (p *PlayerTable) Save(ctx context.Context) error {
	return deprecatedRepos().Tables.SavePlayerTable(ctx, p)
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
