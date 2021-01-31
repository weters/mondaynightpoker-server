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
	case ActionBetTheGap:
		return "Bet the Gap"
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
	participant := a.getCurrentTurn()
	if playerID != participant.PlayerID {
		return nil
	}

	switch a.currentRound.State {
	case RoundStateStart:
		// no-op
	case RoundStateFirstCardDealt:
		// no-op

	case RoundStatePendingAceDecision:
		return []Action{ActionPickAceLow, ActionPickAceHigh}

	case RoundStatePendingBet:
		if a.currentRound.canBetTheGap() {
			return []Action{ActionBet, ActionBetTheGap}
		}

		return []Action{ActionBet}
	case RoundStateGameOver:
		// no-op

	case RoundStateRoundOver:
		// no-op
	}

	return nil
}
