package room

import (
	"context"

	"mondaynightpoker-server/pkg/model"
)

// TableStore is the persistence surface the room requires. *model.Repositories
// satisfies it.
type TableStore interface {
	GetPlayerByID(ctx context.Context, id int64) (*model.Player, error)
	GetPlayerTable(ctx context.Context, player *model.Player, table *model.Table) (*model.PlayerTable, error)
	SavePlayerTable(ctx context.Context, playerTable *model.PlayerTable) error
	GetPlayers(ctx context.Context, table *model.Table) ([]*model.PlayerTable, error)
	GetActivePlayersShifted(ctx context.Context, table *model.Table) ([]*model.PlayerTable, error)
	CreateGame(ctx context.Context, table *model.Table, gameType string) (*model.Game, error)
	EndGame(ctx context.Context, game *model.Game, log interface{}, balanceAdjustments map[int64]int) error
}
