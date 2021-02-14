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
