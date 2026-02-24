package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

const (
	actionCheck    = "check"
	actionCall     = "call"
	actionFold     = "fold"
	actionBet      = "bet"
	actionRaise    = "raise"
	actionDiscard  = "discard"
	actionPlayCard = "playCard"
	actionDecide   = "decide"
	actionTrade    = "trade"
)

func cryptoIntn(n int) int {
	if n <= 0 {
		return 0
	}
	val, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(val.Int64())
}

func cryptoFloat64() float64 {
	val, _ := rand.Int(rand.Reader, big.NewInt(1000))
	return float64(val.Int64()) / 1000.0
}

// AutoPilotAction selects an action for the bot to take based on the game state.
// Returns the action string and additionalData map, or empty string if no action.
func AutoPilotAction(gs *GameState) (action string, additionalData map[string]interface{}) {
	if len(gs.ValidActions) == 0 {
		return "", nil
	}

	// Random delay to simulate thinking
	delay := time.Duration(500+cryptoIntn(1500)) * time.Millisecond
	time.Sleep(delay)

	switch gs.GameName {
	case "texas-hold-em", "texas-hold-em-plo", "seven-card", "little-l":
		return pokerAutoPilot(gs)
	case "bourre":
		return bourreAutoPilot(gs)
	case "guts":
		return gutsAutoPilot(gs)
	case "pass-the-poop", "acey-deucey":
		return genericAutoPilot(gs)
	default:
		return genericAutoPilot(gs)
	}
}

func pokerAutoPilot(gs *GameState) (string, map[string]interface{}) {
	// Build a map of available actions
	actionMap := make(map[string]ValidAction)
	for _, a := range gs.ValidActions {
		actionMap[a.Action] = a
	}

	roll := cryptoFloat64()

	// 60% check/call, 25% min bet/raise, 10% random bet, 5% fold
	switch {
	case roll < 0.60:
		if _, ok := actionMap[actionCheck]; ok {
			return actionCheck, nil
		}
		if _, ok := actionMap[actionCall]; ok {
			return actionCall, nil
		}
		return pickBetAction(gs, actionMap, false)
	case roll < 0.85:
		return pickBetAction(gs, actionMap, false)
	case roll < 0.95:
		return pickBetAction(gs, actionMap, true)
	default:
		if _, ok := actionMap[actionFold]; ok {
			return actionFold, nil
		}
		if _, ok := actionMap[actionCheck]; ok {
			return actionCheck, nil
		}
		return gs.ValidActions[0].Action, nil
	}
}

func pickBetAction(gs *GameState, actionMap map[string]ValidAction, randomAmount bool) (string, map[string]interface{}) {
	betAction := ""
	if _, ok := actionMap[actionBet]; ok {
		betAction = actionBet
	} else if _, ok := actionMap[actionRaise]; ok {
		betAction = actionRaise
	}

	if betAction != "" && gs.MinBet > 0 {
		amount := gs.MinBet
		if randomAmount && gs.MaxBet > gs.MinBet {
			amount = gs.MinBet + cryptoIntn(gs.MaxBet-gs.MinBet+1)
		}
		return betAction, map[string]interface{}{
			"amount": amount,
		}
	}

	if _, ok := actionMap[actionCheck]; ok {
		return actionCheck, nil
	}
	if _, ok := actionMap[actionCall]; ok {
		return actionCall, nil
	}

	return gs.ValidActions[0].Action, nil
}

func bourreAutoPilot(gs *GameState) (string, map[string]interface{}) {
	if len(gs.ValidActions) == 0 {
		return "", nil
	}

	a := gs.ValidActions[0]

	if a.Action == actionDiscard {
		return actionDiscard, map[string]interface{}{
			"cards": []interface{}{},
		}
	}

	if a.Action == actionPlayCard {
		choice := gs.ValidActions[cryptoIntn(len(gs.ValidActions))]
		if len(choice.Cards) > 0 {
			c := choice.Cards[0]
			return actionPlayCard, map[string]interface{}{
				"cards": []map[string]interface{}{
					{"rank": c.Rank, "suit": c.Suit},
				},
			}
		}
	}

	return a.Action, nil
}

func gutsAutoPilot(gs *GameState) (string, map[string]interface{}) {
	for _, a := range gs.ValidActions {
		switch a.Action {
		case actionDecide:
			goIn := cryptoFloat64() < 0.5
			return actionDecide, map[string]interface{}{
				"decision": goIn,
			}
		case "decide-out":
			continue
		case actionTrade:
			return actionTrade, map[string]interface{}{
				"cards": []interface{}{},
			}
		}
	}

	if len(gs.ValidActions) > 0 {
		return gs.ValidActions[0].Action, nil
	}
	return "", nil
}

func genericAutoPilot(gs *GameState) (string, map[string]interface{}) {
	if len(gs.ValidActions) == 0 {
		return "", nil
	}

	choice := gs.ValidActions[cryptoIntn(len(gs.ValidActions))]
	ad := make(map[string]interface{})
	ad["id"] = choice.Action

	var intID int
	if _, err := fmt.Sscanf(choice.Action, "%d", &intID); err == nil {
		return choice.Action, ad
	}

	return choice.Action, nil
}
