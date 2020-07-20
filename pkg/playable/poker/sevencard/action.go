package sevencard

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Action is an action that a player can take in game
type Action string

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

// Action constants
const (
	ActionFold    Action = "fold"
	ActionCheck   Action = "check"
	ActionBet     Action = "bet"
	ActionRaise   Action = "raise"
	ActionCall    Action = "call"
	ActionEndGame Action = "end-game"
)

var allowedActions = map[Action]bool{
	ActionFold:    true,
	ActionCheck:   true,
	ActionBet:     true,
	ActionRaise:   true,
	ActionCall:    true,
	ActionEndGame: true,
}

func (a Action) String() string {
	if a == ActionEndGame {
		return "End Game"
	}

	if _, ok := allowedActions[a]; !ok {
		panic(fmt.Sprintf("unknown action: %s", string(a)))
	}

	raw := []rune(a)
	return strings.ToTitle(string(raw[0])) + string(raw[1:])
}

// ActionFromString returns an action from a string
// If the action isn't known, an error is returned
func ActionFromString(a string) (Action, error) {
	action := Action(a)
	if _, ok := allowedActions[action]; ok {
		return action, nil
	}

	return "", fmt.Errorf("unknown action: %s", a)
}

func (g *Game) getActionsForParticipant(p *participant) []Action {
	actions := make([]Action, 0)
	if g.getCurrentTurn() == p {
		if g.currentBet == 0 {
			actions = append(actions, ActionFold, ActionCheck, ActionBet)
		} else {
			actions = append(actions, ActionFold, ActionCall, ActionRaise)
		}
	}

	if g.isGameOver() {
		actions = append(actions, ActionEndGame)
	}

	return actions
}
