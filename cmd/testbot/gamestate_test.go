package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGameState_passThePoop(t *testing.T) {
	data := json.RawMessage(`{
		"gameState": {"currentTurn": 7, "pot": 300},
		"card": {"rank": 13, "suit": "hearts"},
		"availableActions": [
			{"id": "stay", "name": "Stay"},
			{"id": "flip-king", "name": "Flip King"}
		]
	}`)

	gs, err := ParseGameState(gamePassThePoop, data, 7)
	assert.NoError(t, err)
	assert.Equal(t, gamePassThePoop, gs.GameName)
	assert.Equal(t, int64(7), gs.CurrentTurn)
	assert.Equal(t, 300, gs.Pot)
	assert.Equal(t, []CardInfo{{Rank: 13, Suit: "hearts"}}, gs.Hand)
	assert.Equal(t, []ValidAction{
		{Action: "stay", Name: "Stay"},
		{Action: "flip-king", Name: "Flip King"},
	}, gs.ValidActions)
}

func TestParseGameState_aceyDeucey(t *testing.T) {
	data := json.RawMessage(`{
		"gameState": {
			"currentTurn": 3,
			"maxBet": 500,
			"round": {
				"games": [{"firstCard": {"rank": 2, "suit": "clubs"}, "lastCard": {"rank": 14, "suit": "spades"}}],
				"activeGameIndex": 0
			}
		},
		"actions": [
			{"id": "pass", "name": "Pass"},
			{"id": "bet", "name": "Bet"},
			{"id": "bet-the-gap", "name": "Bet the Gap"}
		]
	}`)

	gs, err := ParseGameState(gameAceyDeucey, data, 3)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), gs.CurrentTurn)
	assert.Equal(t, 25, gs.MinBet)
	assert.Equal(t, 500, gs.MaxBet)
	assert.Equal(t, []CardInfo{{Rank: 2, Suit: "clubs"}, {Rank: 14, Suit: "spades"}}, gs.AceyCards)
	assert.Equal(t, []ValidAction{
		{Action: "pass", Name: "Pass"},
		{Action: "bet", Name: "Bet", NeedsAmount: true},
		{Action: "bet-the-gap", Name: "Bet the Gap"},
	}, gs.ValidActions)
}

func TestBuildMessage_stringVerbGames(t *testing.T) {
	poop := &GameState{GameName: gamePassThePoop}
	msg := BuildMessage(poop, ValidAction{Action: "stay", Name: "Stay"}, nil)
	assert.Equal(t, "stay", msg.Action)
	assert.Empty(t, msg.Subject)

	acey := &GameState{GameName: gameAceyDeucey}
	msg = BuildMessage(acey, ValidAction{Action: "bet", Name: "Bet"}, map[string]interface{}{"amount": 100})
	assert.Equal(t, "bet", msg.Action)
	assert.Empty(t, msg.Subject)
	assert.Equal(t, map[string]interface{}{"amount": 100}, msg.AdditionalData)
}
