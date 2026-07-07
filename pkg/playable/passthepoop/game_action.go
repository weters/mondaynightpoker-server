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
		ID   string `json:"id"`
		Name string `json:"name"`
	}{
		ID:   g.ID(),
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
	// ActionBlockTrade happens when the preceding player has attempted to trade with the player,
	// and the player plays the block chip
	ActionBlockTrade
	// ActionGoToDeck happens when the dealer announces their intention to go to the deck
	ActionGoToDeck
	ActionDrawFromDeck
)

// GameActionFromID returns a GameAction object from its string identifier
func GameActionFromID(id string) (GameAction, error) {
	for action := ActionStay; action <= ActionDrawFromDeck; action++ {
		if action.ID() == id {
			return action, nil
		}
	}

	return 0, fmt.Errorf("no action with identifier %v", id)
}

// ID returns the client-facing string identifier of the GameAction
func (g GameAction) ID() string {
	switch g {
	case ActionStay:
		return "stay"
	case ActionTrade:
		return "trade"
	case ActionAccept:
		return "accept-trade"
	case ActionFlipKing:
		return "flip-king"
	case ActionBlockTrade:
		return "block-trade"
	case ActionGoToDeck:
		return "go-to-deck"
	case ActionDrawFromDeck:
		return "draw-from-deck"
	}

	panic(fmt.Sprintf("invalid action %d", g))
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
	case ActionBlockTrade:
		return "Block Trade"
	case ActionGoToDeck:
		return "Go to Deck"
	case ActionDrawFromDeck:
		return "Draw Card from Deck"
	}

	panic(fmt.Sprintf("invalid action %d", g))
}
