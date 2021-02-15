package texasholdem

import (
	"fmt"
)

// Action is an action a player can take
type Action struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
}

const (
	checkKey = "Check"
	callKey  = "Call"
	betKey   = "Bet"
	raiseKey = "Raise"
	foldKey  = "Fold"
)

var validActions = map[string]bool{
	checkKey: true,
	callKey:  true,
	betKey:   true,
	raiseKey: true,
	foldKey:  true,
}

var actionCheck = Action{Name: checkKey}
var actionFold = Action{Name: foldKey}

func newAction(name string, amount int) (Action, error) {
	if _, ok := validActions[name]; !ok {
		return Action{}, fmt.Errorf("%s is not a valid action", name)
	}

	return Action{
		Name:   name,
		Amount: amount,
	}, nil
}

func mustAction(action Action, err error) Action {
	if err != nil {
		panic(err)
	}

	return action
}

// LogString returns the string representation destined for the client log
func (a Action) LogString() string {
	switch a.Name {
	case checkKey:
		fallthrough
	case foldKey:
		return a.Name
	default:
		return fmt.Sprintf("%s ${%d}", a.Name, a.Amount)
	}
}

// IsZero will return true if it's valid
func (a Action) IsZero() bool {
	return a.Name == ""
}
