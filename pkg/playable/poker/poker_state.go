package poker

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/potmanager"
)

// State provides the current state data for common poker values
type State struct {
	Ante       int             `json:"ante"`
	CurrentBet int             `json:"currentBet"`
	MinBet     int             `json:"minBet"`
	MaxBet     int             `json:"maxBet"`
	Pots       potmanager.Pots `json:"pots"`
	Community  deck.Hand       `json:"community"`
}
