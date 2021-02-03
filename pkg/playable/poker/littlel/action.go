package littlel

import (
	"encoding/json"
	"fmt"
)

// Action represents an action a player can take
type Action string

// action constants
const (
	ActionTrade Action = "trade"
	ActionFold  Action = "fold"
	ActionCheck Action = "check"
	ActionCall  Action = "call"
	ActionBet   Action = "bet"
	ActionRaise Action = "raise"
)

var allowedActions = map[Action]bool{
	ActionTrade: true,
	ActionFold:  true,
	ActionCheck: true,
	ActionCall:  true,
	ActionBet:   true,
	ActionRaise: true,
}

// ActionFromString returns an action for the given string
func ActionFromString(s string) (Action, error) {
	if _, ok := allowedActions[Action(s)]; ok {
		return Action(s), nil
	}

	return "", fmt.Errorf("unknown action for identifier: %s", s)
}

func (a Action) String() string {
	switch a {
	case ActionTrade:
		return "Trade"
	case ActionFold:
		return "Fold"
	case ActionCheck:
		return "Check"
	case ActionCall:
		return "Call"
	case ActionBet:
		return "Bet"
	case ActionRaise:
		return "Raise"
	}

	panic("unknown action")
}

// MarshalJSON encodes the action into JSON
func (a Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{
		ID:   string(a),
		Name: a.String(),
	})
}
