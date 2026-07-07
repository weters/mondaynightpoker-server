package aceydeucey

import (
	"encoding/json"
	"fmt"
)

// Action is an action a participant can take when it's their turn
type Action int

// MarshalJSON encodes the JSON
func (a Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{
		ID:   a.ID(),
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

// ID returns the client-facing string identifier of the Action
func (a Action) ID() string {
	switch a {
	case ActionPending:
		return "pending"
	case ActionPickAceLow:
		return "pick-ace-low"
	case ActionPickAceHigh:
		return "pick-ace-high"
	case ActionBet:
		return "bet"
	case ActionBetTheGap:
		return "bet-the-gap"
	case ActionPass:
		return "pass"
	}

	panic(fmt.Sprintf("invalid action: %d", a))
}

// ActionFromID returns an action from its string identifier
func ActionFromID(id string) (Action, error) {
	for action := ActionPending; action <= ActionPass; action++ {
		if action.ID() == id {
			return action, nil
		}
	}

	return -1, fmt.Errorf("invalid action: %s", id)
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
