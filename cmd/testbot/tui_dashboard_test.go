package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderDashboard(t *testing.T) {
	bots := []*Bot{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob", AutoPilot: true},
	}
	bots[0].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
		Hand:         []CardInfo{{Rank: 14, Suit: "hearts"}, {Rank: 13, Suit: "spades"}},
		Balance:      1500,
	}

	out := RenderDashboard(bots, 0)
	assert.Contains(t, out, "p1 Alice")
	assert.Contains(t, out, "p2 Bob")
	assert.Contains(t, out, "ACT")
	assert.Contains(t, out, "auto")
	assert.Contains(t, out, "$15")
	assert.Contains(t, out, "A♥")
	assert.Contains(t, out, "K♠")
	assert.Contains(t, out, "▸") // focus marker
}

func TestBotStatus(t *testing.T) {
	b := &Bot{ID: 1, Name: "Alice"}

	status, _ := botStatus(b, nil)
	assert.Equal(t, "idle", status)

	gs := &GameState{ValidActions: []ValidAction{{Action: "check"}}}
	status, _ = botStatus(b, gs)
	assert.Equal(t, "ACT", status)

	b.AutoPilot = true
	status, _ = botStatus(b, gs)
	assert.Equal(t, "auto", status)

	b.disconnected = true
	status, _ = botStatus(b, gs)
	assert.Equal(t, "off", status)
}

func TestRenderHelp(t *testing.T) {
	out := RenderHelp(100, 40)
	assert.Contains(t, out, "Keys")
	assert.Contains(t, out, "dashboard")
	assert.Contains(t, out, "auto-pilot")
	assert.Contains(t, out, "terminate")
}
