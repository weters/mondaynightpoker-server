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
	ActionFold  Action = "fold"
	ActionCheck Action = "check"
	ActionBet   Action = "bet"
	ActionRaise Action = "raise"
	ActionCall  Action = "call"
)

// Variant-specific action constants
const (
	ActionFlipMushroom Action = "flip-mushroom"
	ActionPlayAntidote Action = "play-antidote"
)

var allowedActions = map[Action]bool{
	ActionFold:         true,
	ActionCheck:        true,
	ActionBet:          true,
	ActionRaise:        true,
	ActionCall:         true,
	ActionFlipMushroom: true,
	ActionPlayAntidote: true,
}

func (a Action) String() string {
	if _, ok := allowedActions[a]; !ok {
		panic(fmt.Sprintf("unknown action: %s", string(a)))
	}

	// Handle hyphenated actions (e.g., "flip-mushroom" -> "Flip Mushroom")
	parts := strings.Split(string(a), "-")
	for i, part := range parts {
		if len(part) > 0 {
			raw := []rune(part)
			parts[i] = strings.ToTitle(string(raw[0])) + string(raw[1:])
		}
	}
	return strings.Join(parts, " ")
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
	// Don't show betting actions if variant phase is pending
	if iv, ok := g.options.Variant.(InteractiveVariant); ok {
		if iv.IsVariantPhasePending() {
			return make([]Action, 0)
		}
	}

	actions := make([]Action, 0)
	if g.getCurrentTurn() == p {
		if g.currentBet == 0 {
			actions = append(actions, ActionFold, ActionCheck, ActionBet)
		} else {
			actions = append(actions, ActionFold, ActionCall, ActionRaise)
		}
	}

	return actions
}

func (g *Game) getFutureActionsForParticipant(p *participant) []Action {
	// Don't show future betting actions if variant phase is pending
	if iv, ok := g.options.Variant.(InteractiveVariant); ok {
		if iv.IsVariantPhasePending() {
			return make([]Action, 0)
		}
	}

	actions := make([]Action, 0)
	if g.getCurrentTurn() != p && !p.didFold {
		if g.currentBet == 0 {
			actions = append(actions, ActionFold, ActionCheck)
		} else {
			actions = append(actions, ActionFold, ActionCall)
		}
	}

	return actions
}
