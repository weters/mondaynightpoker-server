package main

import (
	"encoding/json"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func newTestModel() Model {
	bots := []*Bot{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}
	return NewModel(bots)
}

func TestNewModel(t *testing.T) {
	m := newTestModel()
	assert.Len(t, m.bots, 3)
	assert.Equal(t, 0, m.active)
	assert.NotNil(t, m.logBuf)
}

func TestModelInit(t *testing.T) {
	m := newTestModel()
	cmd := m.Init()
	assert.Nil(t, cmd)
}

func TestModelWindowSize(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	um := updated.(Model)
	assert.Equal(t, 120, um.width)
	assert.Equal(t, 40, um.height)
}

func TestModelBotStateAutoFocus(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40

	// Bot 2 gets actions
	m.bots[1].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}

	updated, _ := m.Update(BotStateMsg{BotID: 2})
	um := updated.(Model)
	assert.Equal(t, 1, um.active) // should auto-focus to bot index 1
}

func TestModelBotStateNoFocusWhenCurrentHasActions(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40

	// Current bot (index 0) has actions
	m.bots[0].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}

	// Bot 2 also gets actions
	m.bots[1].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}

	updated, _ := m.Update(BotStateMsg{BotID: 2})
	um := updated.(Model)
	assert.Equal(t, 0, um.active) // should stay on current bot
}

func TestModelTabCyclesBots(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	um := updated.(Model)
	assert.Equal(t, 1, um.active)

	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyTab})
	um = updated.(Model)
	assert.Equal(t, 2, um.active)

	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyTab})
	um = updated.(Model)
	assert.Equal(t, 0, um.active) // wraps around
}

func TestModelShiftTabCycles(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	um := updated.(Model)
	assert.Equal(t, 2, um.active) // wraps backward
}

func TestModelEscOpensOverlay(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	um := updated.(Model)
	assert.True(t, um.overlay.Active)
}

func TestModelGameLogMsg(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(GameLogMsg{Message: "Alice bets $50"})
	um := updated.(Model)
	entries := um.logBuf.Recent(10)
	assert.Len(t, entries, 1)
	assert.Equal(t, "Alice bets $50", entries[0].Message)
}

func TestModelGameEndedMsg(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(GameEndedMsg{BotID: 1})
	um := updated.(Model)
	entries := um.logBuf.Recent(10)
	assert.Len(t, entries, 1)
	assert.Contains(t, entries[0].Message, "Game ended")
}

func TestModelErrorMsg(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(ErrorMsg{BotID: 1, Message: "something broke"})
	um := updated.(Model)
	assert.Equal(t, "something broke", um.errMsg)
}

func TestModelActionKeyNoState(t *testing.T) {
	m := newTestModel()
	// No game state, pressing 1 should be a no-op
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	um := updated.(Model)
	assert.Equal(t, 0, um.active) // should not change anything
}

func TestModelActionKeyOutOfRange(t *testing.T) {
	m := newTestModel()
	m.bots[0].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}

	// Press 9 (only 1 action available)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	um := updated.(Model)
	assert.Equal(t, 0, um.active) // should not change anything
}

func TestModelBetInputFlow(t *testing.T) {
	m := newTestModel()
	m.bots[0].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "bet", Name: "Bet"}},
		MinBet:       50,
		MaxBet:       300,
	}
	// Mock sendCh to avoid blocking
	m.bots[0].sendCh = make(chan outgoingMessage, 64)
	m.bots[0].done = make(chan struct{})

	// Press 1 for bet
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	um := updated.(Model)
	assert.Equal(t, inputBet, um.inputMode)

	// Escape cancels
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyEscape})
	um = updated.(Model)
	assert.Equal(t, inputNone, um.inputMode)
}

func TestModelViewEmpty(t *testing.T) {
	m := newTestModel()
	assert.Equal(t, "Initializing...", m.View())
}

func TestModelViewWithSize(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24
	view := m.View()
	assert.Contains(t, view, "Alice")
	assert.Contains(t, view, "Waiting for actions")
}

func TestBotIndexByID(t *testing.T) {
	m := newTestModel()
	assert.Equal(t, 0, m.botIndexByID(1))
	assert.Equal(t, 1, m.botIndexByID(2))
	assert.Equal(t, 2, m.botIndexByID(3))
	assert.Equal(t, -1, m.botIndexByID(99))
}

func TestFormatGameName(t *testing.T) {
	assert.Equal(t, "Texas Hold'em", formatGameName(gameTexasHoldEm))
	assert.Equal(t, "Texas Hold'em (PLO)", formatGameName(gameTexasHoldEmPLO))
	assert.Equal(t, "Bourre", formatGameName("bourre"))
	assert.Equal(t, "Guts", formatGameName("guts"))
	assert.Equal(t, "Pass the Poop", formatGameName("pass-the-poop"))
	assert.Equal(t, "Acey Deucey", formatGameName("acey-deucey"))
	assert.Equal(t, "Seven Card", formatGameName(gameSevenCard))
	assert.Equal(t, "Little L", formatGameName(gameLittleL))
	assert.Equal(t, "unknown-game", formatGameName("unknown-game"))
}

func TestParseLogs(t *testing.T) {
	t.Run("array of log objects", func(t *testing.T) {
		data := json.RawMessage(`[{"message":"Alice bets $50"},{"message":"Bob calls"}]`)
		msgs := parseLogs(data)
		assert.Len(t, msgs, 2)
		assert.Equal(t, "Alice bets $50", msgs[0])
		assert.Equal(t, "Bob calls", msgs[1])
	})

	t.Run("single log object", func(t *testing.T) {
		data := json.RawMessage(`{"message":"Game started"}`)
		msgs := parseLogs(data)
		assert.Len(t, msgs, 1)
		assert.Equal(t, "Game started", msgs[0])
	})

	t.Run("empty array", func(t *testing.T) {
		data := json.RawMessage(`[]`)
		msgs := parseLogs(data)
		assert.Empty(t, msgs)
	})

	t.Run("invalid json", func(t *testing.T) {
		data := json.RawMessage(`not json`)
		msgs := parseLogs(data)
		assert.Nil(t, msgs)
	})

	t.Run("empty messages filtered", func(t *testing.T) {
		data := json.RawMessage(`[{"message":""},{"message":"real"}]`)
		msgs := parseLogs(data)
		assert.Len(t, msgs, 1)
		assert.Equal(t, "real", msgs[0])
	})
}

func TestModelOverlayQuit(t *testing.T) {
	m := newTestModel()

	// Open overlay
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	um := updated.(Model)
	assert.True(t, um.overlay.Active)

	// Navigate to quit (last item)
	for range len(um.overlay.Items) - 1 {
		updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyDown})
		um = updated.(Model)
	}

	// Confirm quit
	updated, cmd := um.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = updated.(Model)
	assert.NotNil(t, cmd) // should return tea.Quit
}

func TestModelStatusBar(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24
	bar := m.renderStatusBar(76)
	assert.Contains(t, bar, "p1 Alice")
	assert.Contains(t, bar, "p2 Bob")
	assert.Contains(t, bar, "p3 Charlie")
}
