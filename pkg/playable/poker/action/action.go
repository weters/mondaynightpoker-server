package action

import (
	"encoding/json"
	"fmt"
)

// Action represents an action a player can take
type Action string

// action constants
const (
	Discard Action = "discard"
	Trade   Action = "trade"
	Fold    Action = "fold"
	Check   Action = "check"
	Call    Action = "call"
	Bet     Action = "bet"
	Raise   Action = "raise"
)

var allowedActions = map[Action]bool{
	Discard: true,
	Trade:   true,
	Fold:    true,
	Check:   true,
	Call:    true,
	Bet:     true,
	Raise:   true,
}

// FromString returns an action for the given string
func FromString(s string) (Action, error) {
	if _, ok := allowedActions[Action(s)]; ok {
		return Action(s), nil
	}

	return "", fmt.Errorf("unknown action for identifier: %s", s)
}

func (a Action) String() string {
	switch a {
	case Discard:
		return "Discard"
	case Trade:
		return "Trade"
	case Fold:
		return "Fold"
	case Check:
		return "Check"
	case Call:
		return "Call"
	case Bet:
		return "Bet"
	case Raise:
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

// IsValid returns true if the action is permitted
func (a Action) IsValid() bool {
	_, ok := allowedActions[a]
	return ok
}

// LogMessage returns a message formatted for the log
func (a Action) LogMessage(amount int) string {
	switch a {
	case Discard:
		return "discarded a card"
	case Fold:
		return "folded"
	case Check:
		return "checked"
	case Call:
		return fmt.Sprintf("called ${%d}", amount)
	case Bet:
		return fmt.Sprintf("bet ${%d}", amount)
	case Raise:
		return fmt.Sprintf("raised to ${%d}", amount)
	}

	return ""
}
