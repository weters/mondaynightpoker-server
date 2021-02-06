package aceydeucey

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Action is an action a participant can take when it's their turn
type Action int

// MarshalJSON encodes the JSON
func (a Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		ID:   int(a),
		Name: a.String(),
	})
}

// Action constants
const (
	ActionPending Action = iota
	ActionPickAceLow
	ActionPickAceHigh
	ActionBet
	ActionBetTheGap
	ActionPass
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
	case ActionBetTheGap:
		return "Bet the Gap"
	case ActionPass:
		return "Pass"
	}

	panic(fmt.Sprintf("invalid action: %d", a))
}

// ActionFromString returns an action from a string integer
func ActionFromString(action string) (Action, error) {
	actionInt, err := strconv.Atoi(action)
	if err != nil {
		return 0, err
	}

	if actionInt >= 0 && actionInt <= int(ActionPass) {
		return Action(actionInt), nil
	}

	return -1, fmt.Errorf("invalid action: %s", action)
}

func (g *Game) getActionsForParticipant(playerID int64) []Action {
	participant := g.getCurrentTurn()
	if playerID != participant.PlayerID {
		return nil
	}

	currentRound := g.getCurrentRound()
	switch currentRound.State {
	case RoundStatePendingAceDecision:
		return []Action{ActionPickAceLow, ActionPickAceHigh}

	case RoundStatePendingBet:
		actions := make([]Action, 0)
		if g.options.AllowPass {
			actions = append(actions, ActionPass)
		}

		if currentRound.canBetTheGap() {
			return append(actions, ActionBet, ActionBetTheGap)
		}

		return append(actions, ActionBet)
	}

	return nil
}
