package room

import (
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
)

type clientStatePlayers struct {
	*model.PlayerTable
	IsConnected  bool `json:"isConnected"`
	IsSeated     bool `json:"isSeated"`
	IsNextPicker bool `json:"isNextPicker"`
}

func newErrorResponse(ctx string, err error) *playable.Response {
	return &playable.Response{
		Key:     "error",
		Value:   err.Error(),
		Context: ctx,
	}
}
