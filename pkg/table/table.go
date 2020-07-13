package table

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/db"
	"time"
)

const tableColumns = `
tables.uuid,
tables.name,
tables.created`

// Table represents a poker table
// A table has many players and can have many games
type Table struct {
	UUID    string    `json:"uuid"`
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
}

// ErrPlayerNotAtTable happens when user is not a member of the table
var ErrPlayerNotAtTable = errors.New("player is not a member of the table")

// CreateTable creates a new table
func (p *Player) CreateTable(ctx context.Context, name string) (*Table, error) {
	tx, err := db.Instance().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	u := uuid.New().String()
	const query = `
INSERT INTO tables (uuid, name)
VALUES ($1, $2)
RETURNING created
`
	var created time.Time
	row := tx.QueryRowContext(ctx, query, u, name)
	if err := row.Scan(&created); err != nil {
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
		UUID:    u,
		Name:    name,
		Created: created,
	}, nil
}

func getTableByRow(row db.Scanner) (*Table, error) {
	var t Table
	if err := row.Scan(&t.UUID, &t.Name, &t.Created); err != nil {
		return nil, err
	}

	return &t, nil
}

// GetTableByUUID returns a table by its UUID
func GetTableByUUID(ctx context.Context, uuid string) (*Table, error) {
	const query = `
SELECT ` + tableColumns + `
FROM tables
WHERE uuid = $1`

	row := db.Instance().QueryRowContext(ctx, query, uuid)
	return getTableByRow(row)
}

// Reload will refresh the data from the database
func (t *Table) Reload(ctx context.Context) error {
	tbl, err := GetTableByUUID(ctx, t.UUID)
	if err != nil {
		return err
	}

	*t = *tbl
	return nil
}

// GetActivePlayersShifted returns all the active players at the table with the players shifted by the number of games
func (t *Table) GetActivePlayersShifted(ctx context.Context) ([]*PlayerTable, error) {
	players, err := t.GetPlayers(ctx)
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

	count, err := t.GetGamesCount(ctx)
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
func (t *Table) GetPlayers(ctx context.Context) ([]*PlayerTable, error) {
	const query = `
SELECT ` + playerColumns + `, ` + playerTableColumns + `
FROM players_tables
INNER JOIN players ON players_tables.player_id = players.id
WHERE players_tables.table_uuid = $1
ORDER BY players_tables.id`

	rows, err := db.Instance().QueryContext(ctx, query, t.UUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*PlayerTable, 0)
	for rows.Next() {
		var p Player
		var pt PlayerTable
		if err := rows.Scan(&p.ID, &p.Email, &p.DisplayName, &p.IsSiteAdmin, &p.passwordHash, &p.Created, &p.Updated,
			&pt.ID, &pt.PlayerID, &pt.TableUUID, &pt.IsTableAdmin, &pt.CanStart, &pt.CanRestart, &pt.CanTerminate,
			&pt.Balance, &pt.Active, &pt.IsBlocked, &pt.Created, &pt.Updated); err != nil {
			return nil, err
		}

		pt.Player = &p
		records = append(records, &pt)
	}

	return records, nil
}

// CreateGame will create a new game for the table
func (t *Table) CreateGame(ctx context.Context, gameType string) (*Game, error) {
	const query = `
INSERT INTO games (parent_id, table_uuid, game_type)
VALUES ($1, $2, $3)
RETURNING ` + gamesColumns

	row := db.Instance().QueryRowContext(ctx, query, nil, t.UUID, gameType)
	return gameByRow(row)
}

// GetGamesCount returns the number of games played by the table
func (t *Table) GetGamesCount(ctx context.Context) (int64, error) {
	const query = `
SELECT COUNT(id)
FROM games
WHERE table_uuid = $1`

	var count int64
	if err := db.Instance().QueryRowContext(ctx, query, t.UUID).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil {
		logrus.WithError(err).Error("could not rollback transaction")
	}
}
