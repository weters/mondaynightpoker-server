package aceydeucey

import (
	"fmt"
	"strconv"
)

// Action is an action a participant can take when it's their turn
type Action int

// Action constants
const (
	ActionPending Action = iota
	ActionPickAceLow
	ActionPickAceHigh
	ActionBet
	ActionPass

	ActionContinue
)

func (a Action) String() string {
	switch a {
	case ActionPending:
		return "Pending"
	case ActionPickAceLow:
		return "Pick Low Ace"
	case ActionPickAceHigh:
		return "Pick High Ace"
	case ActionBet:
		return "Bet"
	case ActionPass:
		return "Pass"
	case ActionContinue:
		return "Continue"
	}

	panic(fmt.Sprintf("invalid action: %d", a))
}

// ActionFromString returns an action from a string integer
func ActionFromString(action string) (Action, error) {
	actionInt, err := strconv.Atoi(action)
	if err != nil {
		return 0, err
	}

	if actionInt >= 0 && actionInt <= int(ActionContinue) {
		return Action(actionInt), nil
	}

	return -1, fmt.Errorf("invalid action: %s", action)
}

func (a *AceyDeucey) getActionsForParticipant(playerID int64) []Action {
	return []Action{}
}
