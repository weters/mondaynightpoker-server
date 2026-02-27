package main

import (
	"crypto/rand"
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

	gameBourre         = "bourre"
	gameGuts           = "guts"
	gamePassThePoop    = "pass-the-poop"
	gameAceyDeucey     = "acey-deucey"
	gameTexasHoldEm    = "texas-hold-em"
	gameTexasHoldEmPLO = "texas-hold-em-plo"
	gameSevenCard      = "seven-card"
	gameLittleL        = "little-l"

	betIncrement = 25
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
// Returns a fully formed outgoingMessage ready to send, or nil if no action.
func AutoPilotAction(gs *GameState) *outgoingMessage {
	if len(gs.ValidActions) == 0 {
		return nil
	}

	// Random delay to simulate thinking
	delay := time.Duration(500+cryptoIntn(1500)) * time.Millisecond
	time.Sleep(delay)

	switch gs.GameName {
	case gameTexasHoldEm, gameTexasHoldEmPLO, gameSevenCard, gameLittleL:
		return pokerAutoPilot(gs)
	case gameBourre:
		return bourreAutoPilot(gs)
	case gameGuts:
		return gutsAutoPilot(gs)
	case gamePassThePoop:
		return passThePoopAutoPilot(gs)
	case gameAceyDeucey:
		return aceyDeuceyAutoPilot(gs)
	default:
		return genericAutoPilot(gs)
	}
}

func pokerAutoPilot(gs *GameState) *outgoingMessage {
	// Build a map of available actions
	actionMap := make(map[string]ValidAction)
	for _, a := range gs.ValidActions {
		actionMap[a.Action] = a
	}

	roll := cryptoFloat64()

	var action string
	var ad map[string]interface{}

	// 60% check/call, 25% min bet/raise, 10% random bet, 5% fold
	switch {
	case roll < 0.60:
		if _, ok := actionMap[actionCheck]; ok {
			action = actionCheck
		} else if _, ok := actionMap[actionCall]; ok {
			action = actionCall
		} else {
			action, ad = pickBetAction(gs, actionMap, false)
		}
	case roll < 0.85:
		action, ad = pickBetAction(gs, actionMap, false)
	case roll < 0.95:
		action, ad = pickBetAction(gs, actionMap, true)
	default:
		if _, ok := actionMap[actionFold]; ok {
			action = actionFold
		} else if _, ok := actionMap[actionCheck]; ok {
			action = actionCheck
		} else {
			action = gs.ValidActions[0].Action
		}
	}

	return &outgoingMessage{
		Action:         action,
		AdditionalData: ad,
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
		maxBet := gs.MaxBet
		// Cap to player's all-in amount to avoid "bet exceeds participant's total"
		if gs.Balance > 0 && gs.Balance < maxBet {
			maxBet = gs.Balance
		}
		// Round down to bet increment
		maxBet = (maxBet / betIncrement) * betIncrement

		// If player can't afford the min bet, fall through to check/call/fold
		if maxBet >= gs.MinBet {
			amount := gs.MinBet
			if randomAmount && maxBet > gs.MinBet {
				steps := (maxBet - gs.MinBet) / betIncrement
				if steps > 0 {
					amount = gs.MinBet + cryptoIntn(steps+1)*betIncrement
				}
			}
			return betAction, map[string]interface{}{
				"amount": amount,
			}
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

func bourreAutoPilot(gs *GameState) *outgoingMessage {
	if len(gs.ValidActions) == 0 {
		return nil
	}

	a := gs.ValidActions[0]

	if a.Action == actionDiscard {
		// Discard nothing (keep hand)
		return &outgoingMessage{
			Action: actionDiscard,
			Cards:  []map[string]interface{}{},
		}
	}

	if a.Action == actionPlayCard {
		choice := gs.ValidActions[cryptoIntn(len(gs.ValidActions))]
		if len(choice.Cards) > 0 {
			c := choice.Cards[0]
			return &outgoingMessage{
				Action: actionPlayCard,
				Cards: []map[string]interface{}{
					{"rank": c.Rank, "suit": c.Suit},
				},
			}
		}
	}

	return &outgoingMessage{Action: a.Action}
}

func gutsAutoPilot(gs *GameState) *outgoingMessage {
	for _, a := range gs.ValidActions {
		switch a.Action {
		case actionDecide:
			goIn := cryptoFloat64() < 0.5
			return &outgoingMessage{
				Action: actionDecide,
				AdditionalData: map[string]interface{}{
					"in": goIn,
				},
			}
		case "decide-out":
			continue
		case actionTrade:
			// Trade no cards (keep hand)
			return &outgoingMessage{
				Action: actionTrade,
				AdditionalData: map[string]interface{}{
					"cards": []string{},
				},
			}
		}
	}

	if len(gs.ValidActions) > 0 {
		return &outgoingMessage{Action: gs.ValidActions[0].Action}
	}
	return nil
}

// passThePoopAutoPilot sends Action="execute" with Subject=actionID.
func passThePoopAutoPilot(gs *GameState) *outgoingMessage {
	if len(gs.ValidActions) == 0 {
		return nil
	}

	choice := gs.ValidActions[cryptoIntn(len(gs.ValidActions))]
	return &outgoingMessage{
		Action:  "execute",
		Subject: choice.Action, // the integer action ID as string
	}
}

// aceyDeuceyAutoPilot sends Subject=actionID (the server reads from Subject).
func aceyDeuceyAutoPilot(gs *GameState) *outgoingMessage {
	if len(gs.ValidActions) == 0 {
		return nil
	}

	choice := gs.ValidActions[cryptoIntn(len(gs.ValidActions))]
	msg := &outgoingMessage{
		Subject: choice.Action, // action ID as string in Subject field
	}

	// ActionBet needs an amount in AdditionalData
	if choice.NeedsAmount && gs.MinBet > 0 {
		amount := gs.MinBet
		if gs.MaxBet > gs.MinBet {
			// Pick a random multiple of 25 between min and max
			steps := (gs.MaxBet - gs.MinBet) / 25
			if steps > 0 {
				amount = gs.MinBet + cryptoIntn(steps+1)*25
			}
		}
		msg.AdditionalData = map[string]interface{}{
			"amount": amount,
		}
	}

	return msg
}

func genericAutoPilot(gs *GameState) *outgoingMessage {
	if len(gs.ValidActions) == 0 {
		return nil
	}

	choice := gs.ValidActions[cryptoIntn(len(gs.ValidActions))]
	return &outgoingMessage{
		Action: choice.Action,
		AdditionalData: map[string]interface{}{
			"id": choice.Action,
		},
	}
}

// BuildMessage constructs the correct outgoingMessage for a given game and action.
// This is used by the REPL for manual play.
func BuildMessage(gs *GameState, action ValidAction, ad map[string]interface{}) outgoingMessage {
	switch gs.GameName {
	case gamePassThePoop:
		return outgoingMessage{
			Action:  "execute",
			Subject: action.Action,
		}
	case gameAceyDeucey:
		msg := outgoingMessage{
			Subject: action.Action,
		}
		if ad != nil {
			msg.AdditionalData = ad
		}
		return msg
	case gameBourre:
		msg := outgoingMessage{
			Action: action.Action,
		}
		if cards, ok := ad["cards"]; ok {
			msg.Cards = cards
		}
		return msg
	case gameGuts:
		return outgoingMessage{
			Action:         action.Action,
			AdditionalData: ad,
		}
	default:
		return outgoingMessage{
			Action:         action.Action,
			AdditionalData: ad,
		}
	}
}

// cardToWireFormat converts a CardInfo to the map format used in Cards field.
func cardToWireFormat(c CardInfo) map[string]interface{} {
	return map[string]interface{}{
		"rank": c.Rank,
		"suit": c.Suit,
	}
}

// cardInfosToDeckStrings converts a slice of CardInfo to deck string format for guts trade.
func cardInfosToDeckStrings(cards []CardInfo) []string {
	result := make([]string, len(cards))
	for i, c := range cards {
		result[i] = c.DeckString()
	}
	return result
}
