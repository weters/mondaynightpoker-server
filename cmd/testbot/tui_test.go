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
	assert.Equal(t, 2, um.lastActorBotID)
}

func TestModelBotStateNoFocusWhenAnotherIsAlreadyActor(t *testing.T) {
	// While bot 1 is the active actor, an out-of-band state update for bot 2
	// (e.g. simultaneous decisions) must not steal focus.
	m := newTestModel()
	m.width = 120
	m.height = 40

	m.bots[0].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}
	updated, _ := m.Update(BotStateMsg{BotID: 1})
	um := updated.(Model)
	assert.Equal(t, 0, um.active)
	assert.Equal(t, 1, um.lastActorBotID)

	// Bot 2 also gains actions while bot 1 still has theirs.
	m.bots[1].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}
	updated, _ = um.Update(BotStateMsg{BotID: 2})
	um = updated.(Model)
	assert.Equal(t, 0, um.active) // stay on bot 1
	assert.Equal(t, 1, um.lastActorBotID)
}

func TestModelTabAwayPersistsForSameActor(t *testing.T) {
	// After auto-focusing to bot 1, the user tabs away to bot 2.
	// Re-receiving the same actor's state must not yank focus back.
	m := newTestModel()
	m.width = 120
	m.height = 40

	m.bots[0].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}
	updated, _ := m.Update(BotStateMsg{BotID: 1})
	um := updated.(Model)
	assert.Equal(t, 0, um.active)

	// User tabs away to bot 2 (index 1).
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyTab})
	um = updated.(Model)
	assert.Equal(t, 1, um.active)

	// Same actor's state arrives again — focus must not move back.
	updated, _ = um.Update(BotStateMsg{BotID: 1})
	um = updated.(Model)
	assert.Equal(t, 1, um.active)
	assert.Equal(t, 1, um.lastActorBotID)
}

func TestModelActorAdvanceOverridesTabAway(t *testing.T) {
	// When the user has tabbed away and the action advances to a new bot,
	// the TUI must snap focus to the new actor.
	m := newTestModel()
	m.width = 120
	m.height = 40

	m.bots[0].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}
	updated, _ := m.Update(BotStateMsg{BotID: 1})
	um := updated.(Model)
	assert.Equal(t, 0, um.active)

	// User tabs to bot 3.
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyTab})
	um = updated.(Model)
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyTab})
	um = updated.(Model)
	assert.Equal(t, 2, um.active)

	// Bot 1 acts: their state arrives with no remaining actions.
	m.bots[0].gameState = &GameState{GameName: gameTexasHoldEm}
	updated, _ = um.Update(BotStateMsg{BotID: 1})
	um = updated.(Model)
	assert.Equal(t, 0, um.lastActorBotID) // actor cleared
	assert.Equal(t, 2, um.active)         // still on bot 3 (no new actor yet)

	// Action advances to bot 2: focus must follow even though user had tabbed.
	m.bots[1].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}
	updated, _ = um.Update(BotStateMsg{BotID: 2})
	um = updated.(Model)
	assert.Equal(t, 1, um.active)
	assert.Equal(t, 2, um.lastActorBotID)
}

func TestModelAutoPilotBotNotActor(t *testing.T) {
	// An auto-pilot bot with actions must not claim the actor slot,
	// since the human at the TUI isn't meant to act for them.
	m := newTestModel()
	m.width = 120
	m.height = 40

	m.bots[1].AutoPilot = true
	m.bots[1].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}
	updated, _ := m.Update(BotStateMsg{BotID: 2})
	um := updated.(Model)
	assert.Equal(t, 0, um.active) // no switch
	assert.Equal(t, 0, um.lastActorBotID)
}

func TestModelActorAdvancesWhenPreviousGoesAutoPilot(t *testing.T) {
	// If the current actor flips to auto-pilot before acting, the next
	// state update from another bot should claim the actor slot.
	m := newTestModel()
	m.width = 120
	m.height = 40

	m.bots[0].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}
	updated, _ := m.Update(BotStateMsg{BotID: 1})
	um := updated.(Model)
	assert.Equal(t, 1, um.lastActorBotID)

	// Bot 1 flips to auto-pilot (e.g., via overlay toggle). No state msg yet.
	um.bots[0].AutoPilot = true

	// A new state for bot 2 arrives — bot 1 should no longer count as the
	// actor, so bot 2 claims it.
	um.bots[1].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}},
	}
	updated, _ = um.Update(BotStateMsg{BotID: 2})
	um = updated.(Model)
	assert.Equal(t, 1, um.active)
	assert.Equal(t, 2, um.lastActorBotID)
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

func TestModelClientStateMsg(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(ClientStateMsg{PlayerNames: map[int64]string{42: "Alice", 99: "Bob"}})
	um := updated.(Model)
	assert.Equal(t, "Alice", um.playerNames[42])
	assert.Equal(t, "Bob", um.playerNames[99])

	// Log with player token replacement
	updated, _ = um.Update(GameLogMsg{Message: "{} bets ${500}", PlayerIDs: []int64{42}})
	um = updated.(Model)
	entries := um.logBuf.Recent(10)
	assert.Len(t, entries, 1)
	assert.Equal(t, "Alice bets $5", entries[0].Message)
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
	m.bots[0].closed = make(chan struct{})

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
		entries := parseLogs(data)
		assert.Len(t, entries, 2)
		assert.Equal(t, "Alice bets $50", entries[0].Message)
		assert.Equal(t, "Bob calls", entries[1].Message)
	})

	t.Run("single log object", func(t *testing.T) {
		data := json.RawMessage(`{"message":"Game started"}`)
		entries := parseLogs(data)
		assert.Len(t, entries, 1)
		assert.Equal(t, "Game started", entries[0].Message)
	})

	t.Run("empty array", func(t *testing.T) {
		data := json.RawMessage(`[]`)
		entries := parseLogs(data)
		assert.Empty(t, entries)
	})

	t.Run("invalid json", func(t *testing.T) {
		data := json.RawMessage(`not json`)
		entries := parseLogs(data)
		assert.Nil(t, entries)
	})

	t.Run("empty messages filtered", func(t *testing.T) {
		data := json.RawMessage(`[{"message":""},{"message":"real"}]`)
		entries := parseLogs(data)
		assert.Len(t, entries, 1)
		assert.Equal(t, "real", entries[0].Message)
	})

	t.Run("with player IDs", func(t *testing.T) {
		data := json.RawMessage(`[{"message":"{} bets ${500}","playerIds":[42]}]`)
		entries := parseLogs(data)
		assert.Len(t, entries, 1)
		assert.Equal(t, "{} bets ${500}", entries[0].Message)
		assert.Equal(t, []int64{42}, entries[0].PlayerIDs)
	})
}

func TestParseClientState(t *testing.T) {
	t.Run("valid client state", func(t *testing.T) {
		data := json.RawMessage(`{
			"42": {"player": {"displayName": "Alice"}, "isConnected": true},
			"99": {"player": {"displayName": "Bob"}, "isConnected": false}
		}`)
		names := parseClientState(data)
		assert.Len(t, names, 2)
		assert.Equal(t, "Alice", names[42])
		assert.Equal(t, "Bob", names[99])
	})

	t.Run("invalid json", func(t *testing.T) {
		names := parseClientState(json.RawMessage(`not json`))
		assert.Nil(t, names)
	})
}

func TestFormatLogMessage(t *testing.T) {
	playerNames := map[int64]string{
		42: "Alice",
		99: "Bob",
	}

	t.Run("replace player name", func(t *testing.T) {
		msg := formatLogMessage("{} bets", []int64{42}, playerNames)
		assert.Equal(t, "Alice bets", msg)
	})

	t.Run("replace multiple player names", func(t *testing.T) {
		msg := formatLogMessage("{} vs {}", []int64{42, 99}, playerNames)
		assert.Equal(t, "Alice, Bob vs Alice, Bob", msg)
	})

	t.Run("replace amount tokens", func(t *testing.T) {
		msg := formatLogMessage("{} bets ${500}", []int64{42}, playerNames)
		assert.Equal(t, "Alice bets $5", msg)
	})

	t.Run("no player IDs", func(t *testing.T) {
		msg := formatLogMessage("Game started", nil, playerNames)
		assert.Equal(t, "Game started", msg)
	})

	t.Run("zero player ID skips replacement", func(t *testing.T) {
		msg := formatLogMessage("{} deals", []int64{0}, playerNames)
		assert.Equal(t, "{} deals", msg)
	})

	t.Run("unknown player ID", func(t *testing.T) {
		msg := formatLogMessage("{} folds", []int64{123}, playerNames)
		assert.Equal(t, "Player(123) folds", msg)
	})

	t.Run("negative amount", func(t *testing.T) {
		msg := formatLogMessage("Lost ${-250}", nil, playerNames)
		assert.Equal(t, "Lost -$2.50", msg)
	})

	t.Run("amount with cents", func(t *testing.T) {
		msg := formatLogMessage("Bet ${150}", nil, playerNames)
		assert.Equal(t, "Bet $1.50", msg)
	})

	t.Run("amount no cents", func(t *testing.T) {
		msg := formatLogMessage("Pot is ${1000}", nil, playerNames)
		assert.Equal(t, "Pot is $10", msg)
	})
}

func TestFormatCents(t *testing.T) {
	assert.Equal(t, "$0", formatCents(0))
	assert.Equal(t, "$1", formatCents(100))
	assert.Equal(t, "$1.50", formatCents(150))
	assert.Equal(t, "$10", formatCents(1000))
	assert.Equal(t, "$0.25", formatCents(25))
	assert.Equal(t, "-$5", formatCents(-500))
	assert.Equal(t, "-$2.50", formatCents(-250))
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

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestModelDashboardToggle(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40

	updated, _ := m.Update(keyRune('d'))
	um := updated.(Model)
	assert.True(t, um.showDashboard)
	assert.Contains(t, um.View(), "Dashboard")

	updated, _ = um.Update(keyRune('d'))
	um = updated.(Model)
	assert.False(t, um.showDashboard)
}

func TestModelHelpToggle(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40

	updated, _ := m.Update(keyRune('?'))
	um := updated.(Model)
	assert.True(t, um.showHelp)
	assert.Contains(t, um.View(), "Keys")

	// Any key dismisses help
	updated, _ = um.Update(keyRune('x'))
	um = updated.(Model)
	assert.False(t, um.showHelp)
}

func TestModelAutoPilotKey(t *testing.T) {
	m := newTestModel()
	m.bots[0].sendCh = make(chan outgoingMessage, 64)
	m.bots[0].closed = make(chan struct{})

	updated, _ := m.Update(keyRune('a'))
	um := updated.(Model)
	assert.True(t, um.bots[0].AutoPilot)

	updated, _ = um.Update(keyRune('a'))
	um = updated.(Model)
	assert.False(t, um.bots[0].AutoPilot)
}

func TestModelAllAutoPilotKey(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(keyRune('A'))
	um := updated.(Model)
	for _, b := range um.bots {
		assert.True(t, b.AutoPilot)
	}

	updated, _ = um.Update(keyRune('A'))
	um = updated.(Model)
	for _, b := range um.bots {
		assert.False(t, b.AutoPilot)
	}
}

func TestModelSpeedKey(t *testing.T) {
	resetSpeed(t)
	m := newTestModel()

	updated, _ := m.Update(keyRune('s'))
	um := updated.(Model)
	assert.Equal(t, speedFast, currentSpeed())

	entries := um.logBuf.Recent(1)
	assert.Contains(t, entries[0].Message, "fast")
}

func TestModelGameSelectKey(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(keyRune('g'))
	um := updated.(Model)
	assert.True(t, um.gameSelect.Active)
}

func TestModelRestartKey(t *testing.T) {
	m := newTestModel()
	m.bots[0].sendCh = make(chan outgoingMessage, 64)
	m.bots[0].closed = make(chan struct{})

	// No game seen yet: error flash, nothing sent
	updated, _ := m.Update(keyRune('r'))
	um := updated.(Model)
	assert.NotEmpty(t, um.errMsg)
	assert.Empty(t, um.bots[0].sendCh)

	// A state update establishes the last game
	um.bots[0].gameState = &GameState{GameName: gameGuts}
	updated, _ = um.Update(BotStateMsg{BotID: 1})
	um = updated.(Model)
	assert.Equal(t, gameGuts, um.lastGame)

	updated, _ = um.Update(keyRune('r'))
	um = updated.(Model)
	msg := <-um.bots[0].sendCh
	assert.Equal(t, "createGame", msg.Action)
	assert.Equal(t, gameGuts, msg.Subject)
}

func TestModelTerminateKey(t *testing.T) {
	m := newTestModel()
	m.bots[0].sendCh = make(chan outgoingMessage, 64)
	m.bots[0].closed = make(chan struct{})

	updated, _ := m.Update(keyRune('T'))
	um := updated.(Model)
	msg := <-um.bots[0].sendCh
	assert.Equal(t, "terminateGame", msg.Action)
}

func TestModelLogScrollKeys(t *testing.T) {
	m := newTestModel()
	for range 30 {
		m.logBuf.Add("entry")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	um := updated.(Model)
	assert.Equal(t, logScrollStep, um.logScroll)

	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	um = updated.(Model)
	assert.Equal(t, 0, um.logScroll)

	// PgDn at the tail stays at zero
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	um = updated.(Model)
	assert.Equal(t, 0, um.logScroll)

	// Scrolling up is clamped to the buffer length
	for range 20 {
		updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		um = updated.(Model)
	}
	assert.Equal(t, 30, um.logScroll)

	// End jumps back to the live tail
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyEnd})
	um = updated.(Model)
	assert.Equal(t, 0, um.logScroll)
}

func TestModelBotConnMsg(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(BotConnMsg{BotID: 2, Connected: false})
	um := updated.(Model)
	entries := um.logBuf.Recent(1)
	assert.Contains(t, entries[0].Message, "connection lost")

	updated, _ = um.Update(BotConnMsg{BotID: 2, Connected: true})
	um = updated.(Model)
	entries = um.logBuf.Recent(1)
	assert.Contains(t, entries[0].Message, "reconnected")
}

func TestModelOverlayTerminate(t *testing.T) {
	m := newTestModel()
	m.bots[0].sendCh = make(chan outgoingMessage, 64)
	m.bots[0].closed = make(chan struct{})

	// Open overlay, move to "Terminate Game" (second item), confirm
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	um := updated.(Model)
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyDown})
	um = updated.(Model)
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um = updated.(Model)

	assert.False(t, um.overlay.Active)
	msg := <-um.bots[0].sendCh
	assert.Equal(t, "terminateGame", msg.Action)
}

func TestGameSelectDigitPick(t *testing.T) {
	gs := NewGameSelect()
	_, gameName := gs.Update(keyRune('3'))
	assert.Equal(t, gs.Items[2].Name, gameName)

	// Out-of-range digit is a no-op
	_, gameName = gs.Update(keyRune('9'))
	assert.Equal(t, "", gameName)
}

func TestModelStatusBarShowsSpeed(t *testing.T) {
	resetSpeed(t)
	m := newTestModel()
	bar := m.renderStatusBar(120)
	assert.Contains(t, bar, "speed:normal")
	assert.Contains(t, bar, "?:help")
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
