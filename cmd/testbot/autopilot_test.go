package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPickBetAction_CapsToBalance(t *testing.T) {
	actionMap := map[string]ValidAction{
		actionBet: {Action: actionBet, Name: "Bet"},
	}

	gs := &GameState{
		MinBet:  50,
		MaxBet:  500,
		Balance: 100, // player only has $100 total
	}

	action, ad := pickBetAction(gs, actionMap, false)
	assert.Equal(t, actionBet, action)
	assert.Equal(t, 50, ad["amount"]) // min bet, capped to 100
}

func TestPickBetAction_RandomCapsToBalance(t *testing.T) {
	actionMap := map[string]ValidAction{
		actionBet: {Action: actionBet, Name: "Bet"},
	}

	gs := &GameState{
		MinBet:  25,
		MaxBet:  500,
		Balance: 75, // player only has $75
	}

	// Run multiple times to check random bets are capped
	for range 100 {
		_, ad := pickBetAction(gs, actionMap, true)
		amount := ad["amount"].(int)
		assert.LessOrEqual(t, amount, 75, "bet should not exceed player balance")
		assert.GreaterOrEqual(t, amount, 25, "bet should be at least min bet")
		assert.Equal(t, 0, amount%betIncrement, "bet should be a multiple of increment")
	}
}

func TestPickBetAction_NoCapWhenBalanceExceedsMax(t *testing.T) {
	actionMap := map[string]ValidAction{
		actionBet: {Action: actionBet, Name: "Bet"},
	}

	gs := &GameState{
		MinBet:  50,
		MaxBet:  200,
		Balance: 1000, // plenty of balance
	}

	for range 100 {
		_, ad := pickBetAction(gs, actionMap, true)
		amount := ad["amount"].(int)
		assert.LessOrEqual(t, amount, 200)
		assert.GreaterOrEqual(t, amount, 50)
	}
}

func TestPickBetAction_ZeroBalanceFallback(t *testing.T) {
	actionMap := map[string]ValidAction{
		actionCheck: {Action: actionCheck, Name: "Check"},
		actionBet:   {Action: actionBet, Name: "Bet"},
	}

	gs := &GameState{
		MinBet:  50,
		MaxBet:  200,
		Balance: 0, // no balance info (e.g., game doesn't provide it)
	}

	// With zero balance, should not cap (balance=0 means no info)
	action, ad := pickBetAction(gs, actionMap, false)
	assert.Equal(t, actionBet, action)
	assert.Equal(t, 50, ad["amount"])
}

func TestPickBetAction_BalanceBelowMinBet(t *testing.T) {
	actionMap := map[string]ValidAction{
		actionBet:  {Action: actionBet, Name: "Bet"},
		actionCall: {Action: actionCall, Name: "Call"},
	}

	gs := &GameState{
		ValidActions: []ValidAction{
			{Action: actionBet, Name: "Bet"},
			{Action: actionCall, Name: "Call"},
		},
		MinBet:  50,
		MaxBet:  500,
		Balance: 17, // can't afford min bet
	}

	// Should fall through to call since balance < min bet
	action, ad := pickBetAction(gs, actionMap, false)
	assert.Equal(t, actionCall, action)
	assert.Nil(t, ad)
}

func TestBourreAutoPilot_StaysWithFaceCards(t *testing.T) {
	gs := &GameState{
		GameName: gameBourre,
		ValidActions: []ValidAction{
			{Action: actionDiscard, Name: "Discard (max 3)"},
			{Action: actionFold, Name: "Fold"},
		},
		Hand: []CardInfo{
			{Rank: 14, Suit: "hearts"},
			{Rank: 13, Suit: "spades"},
		},
		MaxDraw: 3,
	}

	for range 100 {
		msg := bourreAutoPilot(gs)
		assert.NotNil(t, msg)
		assert.Equal(t, actionDiscard, msg.Action, "bourre bot should not fold with face cards")
	}
}

func TestBourreAutoPilot_FoldsWeakHand(t *testing.T) {
	gs := &GameState{
		GameName: gameBourre,
		ValidActions: []ValidAction{
			{Action: actionDiscard, Name: "Discard (max 3)"},
			{Action: actionFold, Name: "Fold"},
		},
		Hand: []CardInfo{
			{Rank: 3, Suit: "hearts"},
			{Rank: 5, Suit: "spades"},
			{Rank: 7, Suit: "diamonds"},
		},
		TrumpCard: &CardInfo{Rank: 10, Suit: "clubs"},
		MaxDraw:   3,
	}

	foldCount := 0
	for range 200 {
		msg := bourreAutoPilot(gs)
		assert.NotNil(t, msg)
		if msg.Action == actionFold {
			foldCount++
		}
	}
	// With 80% fold rate, should fold most of the time
	assert.Greater(t, foldCount, 100, "weak hand should fold frequently")
}

func TestBourreAutoPilot_KeepsTrumpCards(t *testing.T) {
	gs := &GameState{
		GameName: gameBourre,
		ValidActions: []ValidAction{
			{Action: actionDiscard, Name: "Discard (max 3)"},
			{Action: actionFold, Name: "Fold"},
		},
		Hand: []CardInfo{
			{Rank: 14, Suit: "hearts"},  // trump
			{Rank: 3, Suit: "spades"},   // weak non-trump
			{Rank: 5, Suit: "diamonds"}, // weak non-trump
		},
		TrumpCard: &CardInfo{Rank: 10, Suit: "hearts"},
		MaxDraw:   3,
	}

	for range 100 {
		msg := bourreAutoPilot(gs)
		assert.NotNil(t, msg)
		assert.Equal(t, actionDiscard, msg.Action)
		// Should never discard trump card (Ace of hearts)
		if msg.Cards != nil {
			for _, c := range msg.Cards.([]map[string]interface{}) {
				assert.NotEqual(t, "hearts", c["suit"], "should not discard trump cards")
			}
		}
	}
}

func TestBourreAutoPilot_PlayCard(t *testing.T) {
	gs := &GameState{
		GameName: gameBourre,
		ValidActions: []ValidAction{
			{Action: actionPlayCard, Name: "Play K♠", Cards: []CardInfo{{Rank: 13, Suit: "spades"}}},
			{Action: actionPlayCard, Name: "Play 5♠", Cards: []CardInfo{{Rank: 5, Suit: "spades"}}},
		},
		Hand: []CardInfo{
			{Rank: 13, Suit: "spades"},
			{Rank: 5, Suit: "spades"},
		},
		TrumpCard: &CardInfo{Rank: 10, Suit: "hearts"},
	}

	// Leading: should play highest card
	msg := bourreAutoPilot(gs)
	assert.NotNil(t, msg)
	assert.Equal(t, actionPlayCard, msg.Action)
}

func TestPokerAutoPilot_StrongHandBetsMore(t *testing.T) {
	strongGS := &GameState{
		GameName: gameTexasHoldEm,
		ValidActions: []ValidAction{
			{Action: actionCheck, Name: "Check"},
			{Action: actionBet, Name: "Bet"},
			{Action: actionFold, Name: "Fold"},
		},
		Hand: []CardInfo{
			{Rank: 14, Suit: "hearts"},
			{Rank: 14, Suit: "spades"},
		},
		Community: []CardInfo{
			{Rank: 14, Suit: "clubs"},
			{Rank: 14, Suit: "diamonds"},
			{Rank: 10, Suit: "hearts"},
		},
		MinBet:  25,
		MaxBet:  500,
		Balance: 1000,
	}

	weakGS := &GameState{
		GameName: gameTexasHoldEm,
		ValidActions: []ValidAction{
			{Action: actionCheck, Name: "Check"},
			{Action: actionBet, Name: "Bet"},
			{Action: actionFold, Name: "Fold"},
		},
		Hand: []CardInfo{
			{Rank: 2, Suit: "hearts"},
			{Rank: 7, Suit: "spades"},
		},
		Community: []CardInfo{
			{Rank: 14, Suit: "clubs"},
			{Rank: 13, Suit: "diamonds"},
			{Rank: 10, Suit: "hearts"},
		},
		MinBet:  25,
		MaxBet:  500,
		Balance: 1000,
	}

	strongBetCount := 0
	weakBetCount := 0
	iterations := 500

	for range iterations {
		msg := pokerAutoPilot(strongGS)
		if msg.Action == actionBet {
			strongBetCount++
		}
	}
	for range iterations {
		msg := pokerAutoPilot(weakGS)
		if msg.Action == actionBet {
			weakBetCount++
		}
	}

	assert.Greater(t, strongBetCount, weakBetCount, "strong hands should bet more often than weak hands")
}

func TestGutsGoInDecision_PairGoesInMore(t *testing.T) {
	pairHand := []CardInfo{
		{Rank: 10, Suit: "hearts"},
		{Rank: 10, Suit: "spades"},
	}
	weakHand := []CardInfo{
		{Rank: 3, Suit: "hearts"},
		{Rank: 5, Suit: "spades"},
	}

	pairIn := 0
	weakIn := 0
	for range 500 {
		if gutsGoInDecision(pairHand) {
			pairIn++
		}
		if gutsGoInDecision(weakHand) {
			weakIn++
		}
	}
	assert.Greater(t, pairIn, weakIn, "pair should go in more often than weak hand")
}

func TestGutsTradeDecision_TradesNonPairCard(t *testing.T) {
	hand := []CardInfo{
		{Rank: 10, Suit: "hearts"},
		{Rank: 10, Suit: "spades"},
		{Rank: 3, Suit: "clubs"},
	}
	trade := gutsTradeDecision(hand)
	assert.Len(t, trade, 1)
	assert.Equal(t, 3, trade[0].Rank, "should trade the non-pair card")
}

func TestGutsTradeDecision_NoPairTradesLowest(t *testing.T) {
	hand := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 10, Suit: "spades"},
		{Rank: 3, Suit: "clubs"},
	}
	trade := gutsTradeDecision(hand)
	assert.Len(t, trade, 1)
	assert.Equal(t, 3, trade[0].Rank, "should trade the lowest card")
}

func TestPassThePoopAutoPilot_KingAlwaysStays(t *testing.T) {
	gs := &GameState{
		GameName: gamePassThePoop,
		ValidActions: []ValidAction{
			{Action: "0", Name: "Stay"},
			{Action: "1", Name: "Trade"},
		},
		Hand: []CardInfo{{Rank: 13, Suit: "hearts"}},
	}

	for range 100 {
		msg := passThePoopAutoPilot(gs)
		assert.Equal(t, "0", msg.Subject, "king should always stay")
	}
}

func TestPassThePoopAutoPilot_LowCardTradesOften(t *testing.T) {
	gs := &GameState{
		GameName: gamePassThePoop,
		ValidActions: []ValidAction{
			{Action: "0", Name: "Stay"},
			{Action: "1", Name: "Trade"},
		},
		Hand: []CardInfo{{Rank: 2, Suit: "hearts"}},
	}

	tradeCount := 0
	for range 200 {
		msg := passThePoopAutoPilot(gs)
		if msg.Subject == "1" {
			tradeCount++
		}
	}
	assert.Greater(t, tradeCount, 100, "low card should trade frequently")
}

func TestPassThePoopAutoPilot_ForcedActions(t *testing.T) {
	// Accept (2) should be picked when available
	gs := &GameState{
		GameName: gamePassThePoop,
		ValidActions: []ValidAction{
			{Action: "2", Name: "Accept"},
		},
		Hand: []CardInfo{{Rank: 5, Suit: "hearts"}},
	}

	msg := passThePoopAutoPilot(gs)
	assert.Equal(t, "2", msg.Subject)
}

func TestAceyDeuceyAutoPilot_AcePicksLow(t *testing.T) {
	gs := &GameState{
		GameName: gameAceyDeucey,
		ValidActions: []ValidAction{
			{Action: "1", Name: "Pick Low Ace"},
			{Action: "2", Name: "Pick High Ace"},
		},
	}

	for range 100 {
		msg := aceyDeuceyAutoPilot(gs)
		assert.Equal(t, "1", msg.Subject, "should always pick ace low")
	}
}

func TestAceyDeuceyAutoPilot_LargeGapBetsMore(t *testing.T) {
	largeGapGS := &GameState{
		GameName: gameAceyDeucey,
		ValidActions: []ValidAction{
			{Action: "3", Name: "Bet", NeedsAmount: true},
			{Action: "5", Name: "Pass"},
		},
		AceyCards: []CardInfo{
			{Rank: 2, Suit: "hearts"},
			{Rank: 14, Suit: "spades"},
		},
		MinBet: 25,
		MaxBet: 500,
	}

	smallGapGS := &GameState{
		GameName: gameAceyDeucey,
		ValidActions: []ValidAction{
			{Action: "3", Name: "Bet", NeedsAmount: true},
			{Action: "5", Name: "Pass"},
		},
		AceyCards: []CardInfo{
			{Rank: 7, Suit: "hearts"},
			{Rank: 8, Suit: "spades"},
		},
		MinBet: 25,
		MaxBet: 500,
	}

	largeTotal := 0
	smallTotal := 0
	iterations := 200

	for range iterations {
		msg := aceyDeuceyAutoPilot(largeGapGS)
		if msg.AdditionalData != nil {
			if amount, ok := msg.AdditionalData["amount"].(int); ok {
				largeTotal += amount
			}
		}
	}
	for range iterations {
		msg := aceyDeuceyAutoPilot(smallGapGS)
		if msg.AdditionalData != nil {
			if amount, ok := msg.AdditionalData["amount"].(int); ok {
				smallTotal += amount
			}
		}
	}

	assert.Greater(t, largeTotal, smallTotal, "large gap should result in higher total bets")
}

func TestPickBetAction_BalanceBelowMinBetChecks(t *testing.T) {
	actionMap := map[string]ValidAction{
		actionBet:   {Action: actionBet, Name: "Bet"},
		actionCheck: {Action: actionCheck, Name: "Check"},
	}

	gs := &GameState{
		ValidActions: []ValidAction{
			{Action: actionBet, Name: "Bet"},
			{Action: actionCheck, Name: "Check"},
		},
		MinBet:  50,
		MaxBet:  500,
		Balance: 10, // can't afford min bet
	}

	// Should fall through to check since balance < min bet
	action, ad := pickBetAction(gs, actionMap, false)
	assert.Equal(t, actionCheck, action)
	assert.Nil(t, ad)
}
