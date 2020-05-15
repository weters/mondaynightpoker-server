package bourre

import (
	"errors"
	"fmt"
)

// ErrRoundNotOver is an error when the round is not over yet
var ErrRoundNotOver = errors.New("the round is not over")

// ErrRoundIsOver is an error when cards beyond the round are played
var ErrRoundIsOver = errors.New("the round is over")

// ErrIsNotPlayersTurn is returned when it's not the player's turn
var ErrIsNotPlayersTurn = errors.New("not player's turn")

// ErrCardNotInPlayersHand happens when the player tries to play a card they don't have
var ErrCardNotInPlayersHand = errors.New("card is not in player's hand")

// ErrPlayToWinOnSuit happens when the player doesn't play a winning on-suit card
var ErrPlayToWinOnSuit = errors.New("player has a higher on-suit card")

// ErrPlayToWinOnTrump happens when the player doesn't play a winning trump card
var ErrPlayToWinOnTrump = errors.New("player has a higher trump card")

// ErrPlayOnSuit happens when a player has a suit of the lead suit and plays an off-suit card
var ErrPlayOnSuit = errors.New("player has an on-suit card")

// ErrPlayTrump happens if a player has a trump card and tries to play a non-trump, non lead
var ErrPlayTrump = errors.New("player has a trump card")

// ErrLastPlayerMustPlay prevents the last player left from folding
var ErrLastPlayerMustPlay = errors.New("everyone else folded, so you must play")

// ErrNotEnoughActivePlayers is an error when there are not at least two active players
var ErrNotEnoughActivePlayers = errors.New("need at least two players to continue")

// ErrTradeInRoundInProgress happens if a player tries to play a card before the trade-in round is complete
var ErrTradeInRoundInProgress = errors.New("the trade-in round is not complete")

// ErrTradeInRoundIsOver happens if trade are attempted during the game play
var ErrTradeInRoundIsOver = errors.New("the trade-in round is over")

// ErrGameIsOver is an error when an action is attempted on an ended game
var ErrGameIsOver = errors.New("game is over")

// ErrGameNotOver is an error when someone tries to end the game and it's not over yet
var ErrGameNotOver = errors.New("game is not over")

// ErrCannotDiscardTheSameCard is an error when user has the same card in the/1186 discard array
var ErrCannotDiscardTheSameCard = errors.New("you cannot discard the same card")

// ErrCannotCreateGame happens when you try to create a new game from a result when there's no pot
var ErrCannotCreateGame = errors.New("cannot create a new game from an existing game")

// PlayerCountError is an error on the number of players in the game
type PlayerCountError int

func (p PlayerCountError) Error() string {
	return fmt.Sprintf("expected 2–%d players, got %d", playersLimit, p)
}
