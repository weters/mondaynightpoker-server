package guts

import (
	"errors"
	"fmt"
)

// ErrNotInDeclarationPhase is returned when a decision is made outside the declaration phase
var ErrNotInDeclarationPhase = errors.New("not in declaration phase")

// ErrAlreadyDecided is returned when a player has already made their decision
var ErrAlreadyDecided = errors.New("player has already decided")

// ErrPlayerNotFound is returned when a player is not found in the game
var ErrPlayerNotFound = errors.New("player not found")

// ErrGameIsOver is returned when an action is attempted on an ended game
var ErrGameIsOver = errors.New("game is over")

// ErrNotEnoughPlayers is returned when there aren't enough players
var ErrNotEnoughPlayers = errors.New("need at least two players")

// PlayerCountError is an error on the number of players in the game
type PlayerCountError struct {
	Min int
	Max int
	Got int
}

func (p PlayerCountError) Error() string {
	return fmt.Sprintf("expected %d–%d players, got %d", p.Min, p.Max, p.Got)
}
