package room

import (
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/table"
)

type clientStatePlayers struct {
	*table.PlayerTable
	IsConnected bool `json:"isConnected"`
	IsSeated    bool `json:"isSeated"`
}

func newErrorResponse(ctx string, err error) *playable.Response {
	return &playable.Response{
		Key:     "error",
		Value:   err.Error(),
		Context: ctx,
	}
}
