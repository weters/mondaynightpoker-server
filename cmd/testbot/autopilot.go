package main

import (
	"crypto/rand"
	"math/big"
	"sort"
	"time"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/guts"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
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
	actionMap := make(map[string]ValidAction)
	for _, a := range gs.ValidActions {
		actionMap[a.Action] = a
	}

	var strength float64
	if len(gs.Community) > 0 {
		hand, _ := evaluatePokerHand(gs.Hand, gs.Community)
		strength = handStrengthScore(hand)
	} else {
		strength = startingHandStrength(gs.Hand)
	}

	roll := cryptoFloat64()
	var action string
	var ad map[string]interface{}

	switch {
	case strength < 0.15:
		// Weak hand: 70% fold, 30% check/call
		if roll < 0.70 {
			action = pickPassiveOrFold(gs, actionMap)
		} else {
			action = pickCheckOrCall(actionMap)
		}
	case strength < 0.35:
		// Below average: 60% check/call, 30% min bet, 10% fold
		if roll < 0.60 {
			action = pickCheckOrCall(actionMap)
		} else if roll < 0.90 {
			action, ad = pickBetAction(gs, actionMap, false)
		} else {
			action = pickPassiveOrFold(gs, actionMap)
		}
	case strength < 0.60:
		// Average: 50% check/call, 40% min bet, 10% random bet
		if roll < 0.50 {
			action = pickCheckOrCall(actionMap)
		} else if roll < 0.90 {
			action, ad = pickBetAction(gs, actionMap, false)
		} else {
			action, ad = pickBetAction(gs, actionMap, true)
		}
	case strength < 0.80:
		// Good hand: 30% check/call, 50% bet, 20% larger bet
		if roll < 0.30 {
			action = pickCheckOrCall(actionMap)
		} else if roll < 0.80 {
			action, ad = pickBetAction(gs, actionMap, false)
		} else {
			action, ad = pickBetAction(gs, actionMap, true)
		}
	default:
		// Strong hand: 10% check/call, 40% mid bet, 50% large bet
		if roll < 0.10 {
			action = pickCheckOrCall(actionMap)
		} else if roll < 0.50 {
			action, ad = pickBetAction(gs, actionMap, false)
		} else {
			action, ad = pickBetAction(gs, actionMap, true)
		}
	}

	if action == "" {
		action = gs.ValidActions[0].Action
	}

	return &outgoingMessage{
		Action:         action,
		AdditionalData: ad,
	}
}

// pickCheckOrCall returns check if available, otherwise call, otherwise first action.
func pickCheckOrCall(actionMap map[string]ValidAction) string {
	if _, ok := actionMap[actionCheck]; ok {
		return actionCheck
	}
	if _, ok := actionMap[actionCall]; ok {
		return actionCall
	}
	return ""
}

// pickPassiveOrFold folds if possible, otherwise checks/calls.
func pickPassiveOrFold(gs *GameState, actionMap map[string]ValidAction) string {
	if _, ok := actionMap[actionFold]; ok {
		return actionFold
	}
	if _, ok := actionMap[actionCheck]; ok {
		return actionCheck
	}
	return gs.ValidActions[0].Action
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
		return bourreDiscardDecision(gs)
	}

	if a.Action == actionPlayCard {
		return bourrePlayCard(gs)
	}

	return &outgoingMessage{Action: a.Action}
}

// bourreDiscardDecision decides whether to fold or play, and which cards to discard.
func bourreDiscardDecision(gs *GameState) *outgoingMessage {
	trumpSuit := ""
	if gs.TrumpCard != nil {
		trumpSuit = gs.TrumpCard.Suit
	}

	trumpCount := 0
	faceCount := 0
	for _, c := range gs.Hand {
		if c.Suit == trumpSuit {
			trumpCount++
		}
		if c.Rank >= 11 {
			faceCount++
		}
	}

	// Fold decision: in bourré, folding is done by discarding nil cards
	if trumpCount == 0 && faceCount == 0 {
		if cryptoFloat64() < 0.80 {
			return &outgoingMessage{
				Action: actionDiscard,
				Cards:  nil,
			}
		}
	}

	// Build discard list: weakest non-trump cards
	type rankedCard struct {
		card CardInfo
		idx  int
	}
	var nonTrump []rankedCard
	for i, c := range gs.Hand {
		if c.Suit != trumpSuit {
			nonTrump = append(nonTrump, rankedCard{card: c, idx: i})
		}
	}
	sort.Slice(nonTrump, func(i, j int) bool {
		return nonTrump[i].card.Rank < nonTrump[j].card.Rank
	})

	maxDiscard := gs.MaxDraw
	if maxDiscard <= 0 {
		maxDiscard = 0
	}

	var discards []map[string]interface{}
	for i := 0; i < len(nonTrump) && i < maxDiscard; i++ {
		// Only discard weak cards (below face value)
		if nonTrump[i].card.Rank < 11 {
			discards = append(discards, cardToWireFormat(nonTrump[i].card))
		}
	}

	return &outgoingMessage{
		Action: actionDiscard,
		Cards:  discards,
	}
}

// bourrePlayCard picks the best card to play from valid moves.
func bourrePlayCard(gs *GameState) *outgoingMessage {
	if len(gs.ValidActions) == 0 {
		return nil
	}

	trumpSuit := ""
	if gs.TrumpCard != nil {
		trumpSuit = gs.TrumpCard.Suit
	}

	// Collect all valid card plays
	var cards []CardInfo
	for _, va := range gs.ValidActions {
		if len(va.Cards) > 0 {
			cards = append(cards, va.Cards[0])
		}
	}

	if len(cards) == 0 {
		c := gs.ValidActions[0].Cards[0]
		return &outgoingMessage{
			Action: actionPlayCard,
			Cards:  []map[string]interface{}{cardToWireFormat(c)},
		}
	}

	var chosen CardInfo
	if len(gs.PlayedCards) == 0 {
		// Leading: play highest card, prefer trump
		chosen = cards[0]
		for _, c := range cards[1:] {
			cIsTrump := c.Suit == trumpSuit
			chosenIsTrump := chosen.Suit == trumpSuit
			if cIsTrump && !chosenIsTrump {
				chosen = c
			} else if cIsTrump == chosenIsTrump && c.Rank > chosen.Rank {
				chosen = c
			}
		}
	} else {
		// Following: try trump, then high of suit, else dump lowest
		var trumpCards, otherCards []CardInfo
		for _, c := range cards {
			if c.Suit == trumpSuit {
				trumpCards = append(trumpCards, c)
			} else {
				otherCards = append(otherCards, c)
			}
		}

		if len(trumpCards) > 0 {
			// Play lowest trump that might win
			chosen = trumpCards[0]
			for _, c := range trumpCards[1:] {
				if c.Rank < chosen.Rank {
					chosen = c
				}
			}
		} else if len(otherCards) > 0 {
			// Play highest non-trump
			chosen = otherCards[0]
			for _, c := range otherCards[1:] {
				if c.Rank > chosen.Rank {
					chosen = c
				}
			}
		} else {
			chosen = cards[0]
		}
	}

	return &outgoingMessage{
		Action: actionPlayCard,
		Cards:  []map[string]interface{}{cardToWireFormat(chosen)},
	}
}

func gutsAutoPilot(gs *GameState) *outgoingMessage {
	for _, a := range gs.ValidActions {
		switch a.Action {
		case actionDecide:
			goIn := gutsGoInDecision(gs.Hand)
			return &outgoingMessage{
				Action: actionDecide,
				AdditionalData: map[string]interface{}{
					"in": goIn,
				},
			}
		case "decide-out":
			continue
		case actionTrade:
			tradeCards := gutsTradeDecision(gs.Hand)
			return &outgoingMessage{
				Action: actionTrade,
				AdditionalData: map[string]interface{}{
					"cards": cardInfosToDeckStrings(tradeCards),
				},
			}
		}
	}

	if len(gs.ValidActions) > 0 {
		return &outgoingMessage{Action: gs.ValidActions[0].Action}
	}
	return nil
}

// gutsGoInDecision decides whether to go in based on hand quality.
func gutsGoInDecision(hand []CardInfo) bool {
	roll := cryptoFloat64()

	if len(hand) == 2 {
		if hand[0].Rank == hand[1].Rank {
			return roll < 0.90 // Pair: 90% go in
		}
		highCount := 0
		for _, c := range hand {
			if c.Rank >= 10 {
				highCount++
			}
		}
		switch highCount {
		case 2:
			return roll < 0.60
		case 1:
			return roll < 0.30
		default:
			return roll < 0.10
		}
	}

	// 3-card hand: use hand analyzer
	if len(hand) >= 3 {
		cards := cardInfosToDeckCards(hand)
		result := gutsAnalyzeHand(cards)
		switch result {
		case gutsThreeOfAKind:
			return true
		case gutsStraightOrFlush:
			return roll < 0.85
		case gutsPair:
			return roll < 0.60
		default:
			// High card: check if highest card >= Queen
			maxRank := 0
			for _, c := range hand {
				if c.Rank > maxRank {
					maxRank = c.Rank
				}
			}
			if maxRank >= 12 {
				return roll < 0.25
			}
			return roll < 0.10
		}
	}

	return roll < 0.30
}

// Simplified guts hand types for decision making.
const (
	gutsHighCard        = 0
	gutsPair            = 1
	gutsStraightOrFlush = 2
	gutsThreeOfAKind    = 3
)

// gutsAnalyzeHand analyzes a guts hand and returns a simplified hand type.
func gutsAnalyzeHand(cards []*deck.Card) int {
	result := guts.AnalyzeHand(cards)
	switch result.Type {
	case handanalyzer.ThreeOfAKind, handanalyzer.ThreeCardPokerThreeOfAKind:
		return gutsThreeOfAKind
	case handanalyzer.Straight, handanalyzer.ThreeCardPokerStraight, handanalyzer.Flush, handanalyzer.StraightFlush, handanalyzer.RoyalFlush:
		return gutsStraightOrFlush
	case handanalyzer.OnePair:
		return gutsPair
	default:
		return gutsHighCard
	}
}

// gutsTradeDecision decides which cards to trade.
func gutsTradeDecision(hand []CardInfo) []CardInfo {
	if len(hand) == 0 {
		return nil
	}

	// Check for pair
	rankCounts := make(map[int]int)
	for _, c := range hand {
		rankCounts[c.Rank]++
	}

	var pairRank int
	for r, count := range rankCounts {
		if count >= 2 {
			pairRank = r
			break
		}
	}

	if pairRank > 0 {
		// Trade non-pair cards
		var trade []CardInfo
		for _, c := range hand {
			if c.Rank != pairRank {
				trade = append(trade, c)
			}
		}
		return trade
	}

	// No pair: trade the lowest card
	lowest := hand[0]
	for _, c := range hand[1:] {
		if c.Rank < lowest.Rank {
			lowest = c
		}
	}
	return []CardInfo{lowest}
}

// passThePoopAutoPilot sends Action="execute" with Subject=actionID.
// Decisions based on card rank: high cards stay, low cards trade.
func passThePoopAutoPilot(gs *GameState) *outgoingMessage {
	if len(gs.ValidActions) == 0 {
		return nil
	}

	// Build action ID set
	actionIDs := make(map[string]bool)
	for _, a := range gs.ValidActions {
		actionIDs[a.Action] = true
	}

	// Forced/mandatory actions: Accept (2), FlipKing (3), BlockTrade (4)
	for _, forced := range []string{"2", "3", "4"} {
		if actionIDs[forced] {
			return &outgoingMessage{
				Action:  "execute",
				Subject: forced,
			}
		}
	}

	// Stay (0) vs Trade (1) / GoToDeck (5)
	hasStay := actionIDs["0"]
	hasTrade := actionIDs["1"] || actionIDs["5"]

	if hasStay && hasTrade && len(gs.Hand) > 0 {
		rank := gs.Hand[0].Rank
		roll := cryptoFloat64()

		var stayProb float64
		switch {
		case rank >= 13: // King
			stayProb = 1.0
		case rank >= 12: // Queen/Ace(14)
			stayProb = 0.85
		case rank >= 10: // Jack/10
			stayProb = 0.60
		case rank >= 7:
			stayProb = 0.40
		default: // 2-6
			stayProb = 0.15
		}

		if roll < stayProb {
			return &outgoingMessage{Action: "execute", Subject: "0"}
		}
		// Trade or GoToDeck
		if actionIDs["1"] {
			return &outgoingMessage{Action: "execute", Subject: "1"}
		}
		return &outgoingMessage{Action: "execute", Subject: "5"}
	}

	// Fallback: pick first available action
	return &outgoingMessage{
		Action:  "execute",
		Subject: gs.ValidActions[0].Action,
	}
}

// aceyDeuceyAutoPilot sends Subject=actionID (the server reads from Subject).
// Ace decisions always pick low. Bets are proportional to the gap between cards.
func aceyDeuceyAutoPilot(gs *GameState) *outgoingMessage {
	if len(gs.ValidActions) == 0 {
		return nil
	}

	actionIDs := make(map[string]bool)
	for _, a := range gs.ValidActions {
		actionIDs[a.Action] = true
	}

	// Ace decision: always pick low (1=PickAceLow) for widest spread
	if actionIDs["1"] {
		return &outgoingMessage{Subject: "1"}
	}

	// If we have both cards, make a gap-based bet decision
	if len(gs.AceyCards) == 2 {
		return aceyDeuceyBetDecision(gs, actionIDs)
	}

	// Fallback: pick first available
	return &outgoingMessage{Subject: gs.ValidActions[0].Action}
}

// aceyDeuceyBetDecision makes a betting decision based on the gap between the two cards.
func aceyDeuceyBetDecision(gs *GameState, actionIDs map[string]bool) *outgoingMessage {
	r1 := gs.AceyCards[0].Rank
	r2 := gs.AceyCards[1].Rank

	// Handle ace-low: if rank is 1, it's a low ace
	if r1 == 1 {
		r1 = 1
	}
	if r2 == 1 {
		r2 = 1
	}

	high, low := r1, r2
	if r2 > r1 {
		high, low = r2, r1
	}
	gap := high - low - 1
	if gap < 0 {
		gap = 0
	}

	// Small gap: prefer pass
	if gap <= 1 {
		if actionIDs["5"] { // ActionPass
			return &outgoingMessage{Subject: "5"}
		}
		// If can't pass, min bet
		return aceyDeuceyMakeBet(gs, 0.0)
	}

	// Gap-based bet fraction
	var fraction float64
	switch {
	case gap <= 3:
		fraction = 0.0 // min bet
	case gap <= 6:
		fraction = 0.25 + cryptoFloat64()*0.25 // 25-50%
	case gap <= 9:
		fraction = 0.50 + cryptoFloat64()*0.25 // 50-75%
	default:
		fraction = 0.75 + cryptoFloat64()*0.25 // 75-100%
	}

	// ActionBet=3, ActionBetTheGap=4
	if actionIDs["3"] {
		return aceyDeuceyMakeBet(gs, fraction)
	}

	return &outgoingMessage{Subject: gs.ValidActions[0].Action}
}

// aceyDeuceyMakeBet creates a bet message with amount based on fraction of max bet.
func aceyDeuceyMakeBet(gs *GameState, fraction float64) *outgoingMessage {
	amount := gs.MinBet
	if gs.MaxBet > gs.MinBet && fraction > 0 {
		betRange := gs.MaxBet - gs.MinBet
		extra := int(float64(betRange) * fraction)
		extra = (extra / 25) * 25 // round to increment
		amount = gs.MinBet + extra
		if amount > gs.MaxBet {
			amount = gs.MaxBet
		}
	}
	return &outgoingMessage{
		Subject: "3", // ActionBet
		AdditionalData: map[string]interface{}{
			"amount": amount,
		},
	}
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
