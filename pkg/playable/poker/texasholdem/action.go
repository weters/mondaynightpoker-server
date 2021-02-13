package texasholdem

import (
	"encoding/json"
	"fmt"
)

// Action is an action a player can take
type Action int

type actionJSON struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// MarshalJSON marshals the actio JSON
func (a Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(actionJSON{
		ID:   int(a),
		Name: a.String(),
	})
}

// constants for Action
const (
	ActionCheck Action = iota
	ActionCall
	ActionBet
	ActionRaise
	ActionFold
)

func (a Action) String() string {
	switch a {
	case ActionCheck:
		return "Check"
	case ActionCall:
		return "Call"
	case ActionBet:
		return "Bet"
	case ActionRaise:
		return "Raise"
	case ActionFold:
		return "Fold"
	}

	return ""
}

// ActionFromInt returns an action for the given id
func ActionFromInt(a int) (Action, error) {
	if a < 0 || a > int(ActionFold) {
		return 0, fmt.Errorf("no action with id %d", a)
	}

	return Action(a), nil
}
