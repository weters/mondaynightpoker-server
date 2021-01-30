package aceydeucey

import "fmt"

// Action is an action a participant can take when it's their turn
type Action int

// Action constants
const (
	ActionPickAceLow Action = iota
	ActionPickAceHigh
	ActionBet
	ActionPass

	ActionContinue
)

func (a Action) String() string {
	switch a {
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
