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

func TestBourreAutoPilot_NeverFolds(t *testing.T) {
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
	}

	for range 100 {
		msg := bourreAutoPilot(gs)
		assert.NotNil(t, msg)
		assert.Equal(t, actionDiscard, msg.Action, "bourre bot should never fold")
	}
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
