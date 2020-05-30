package passthepoop

import (
	"encoding/json"
	"fmt"
)

// GameAction is a game action a player can take (i.e., stay or trade)
type GameAction int

// MarshalJSON encodes a GameAction into a JSON object
func (g GameAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		ID:   int(g),
		Name: g.String(),
	})
}

// game action constants
const (
	ActionStay GameAction = iota
	ActionTrade
	// ActionAccept is when the player has to accept the swap from the previous player
	ActionAccept
	// ActionFlipKing is the action a player can take when they have a king and the previous
	// player is attempting to swap
	ActionFlipKing
	// ActionGoToDeck happens when the dealer announces their intention to go to the deck
	ActionGoToDeck
	ActionDrawFromDeck

	ActionEndRound
	ActionNextRound
)

// GameActionFromInt returns a GameAction object from an integer
func GameActionFromInt(i int) (GameAction, error) {
	if i >= 0 && i <= int(ActionNextRound) {
		return GameAction(i), nil
	}

	return 0, fmt.Errorf("no action with identifier %v", i)
}

func (g GameAction) String() string {
	switch g {
	case ActionStay:
		return "Stay"
	case ActionTrade:
		return "Trade"
	case ActionAccept:
		return "Accept Trade"
	case ActionFlipKing:
		return "Flip King"
	case ActionGoToDeck:
		return "Go to Deck"
	case ActionDrawFromDeck:
		return "Draw Card from Deck"
	case ActionEndRound:
		return "End Round"
	case ActionNextRound:
		return "Next Round"
	}

	panic(fmt.Sprintf("invalid action %d", g))
}
