package model

import "context"

// The flat delegates below let *Repositories satisfy narrow consumer interfaces
// (e.g. room.TableStore) without an adapter type.

// GetPlayerByID returns the player by their ID
func (r *Repositories) GetPlayerByID(ctx context.Context, id int64) (*Player, error) {
	return r.Players.GetPlayerByID(ctx, id)
}

// GetPlayerTable returns the player's status at a table
func (r *Repositories) GetPlayerTable(ctx context.Context, player *Player, table *Table) (*PlayerTable, error) {
	return r.Tables.GetPlayerTable(ctx, player, table)
}

// SavePlayerTable persists the player's status at a table
func (r *Repositories) SavePlayerTable(ctx context.Context, playerTable *PlayerTable) error {
	return r.Tables.SavePlayerTable(ctx, playerTable)
}

// GetPlayers returns the players at a table
func (r *Repositories) GetPlayers(ctx context.Context, table *Table) ([]*PlayerTable, error) {
	return r.Tables.GetPlayers(ctx, table)
}

// GetActivePlayersShifted returns the active players in rotated seat order
func (r *Repositories) GetActivePlayersShifted(ctx context.Context, table *Table) ([]*PlayerTable, error) {
	return r.Tables.GetActivePlayersShifted(ctx, table)
}

// CreateGame creates a game record for a table
func (r *Repositories) CreateGame(ctx context.Context, table *Table, gameType string) (*Game, error) {
	return r.Games.CreateGame(ctx, table, gameType)
}

// EndGame finalizes a game record and applies balance adjustments
func (r *Repositories) EndGame(ctx context.Context, game *Game, log interface{}, balanceAdjustments map[int64]int) error {
	return r.Games.EndGame(ctx, game, log, balanceAdjustments)
}
