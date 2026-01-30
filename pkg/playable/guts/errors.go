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

// ErrNotInTradePhase is returned when a trade is attempted outside the trade phase
var ErrNotInTradePhase = errors.New("not in trade phase")

// ErrNotYourTurnToTrade is returned when a player tries to trade out of turn
var ErrNotYourTurnToTrade = errors.New("not your turn to trade")

// ErrInvalidTradeCount is returned when trading more cards than allowed
var ErrInvalidTradeCount = errors.New("invalid number of cards to trade")

// ErrCardNotInHand is returned when trying to trade a card not in hand
var ErrCardNotInHand = errors.New("card not in hand")

// PlayerCountError is an error on the number of players in the game
type PlayerCountError struct {
	Min int
	Max int
	Got int
}

func (p PlayerCountError) Error() string {
	return fmt.Sprintf("expected %d–%d players, got %d", p.Min, p.Max, p.Got)
}
