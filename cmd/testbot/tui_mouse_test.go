package main

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain initializes the global bubblezone manager used for mouse hit-testing
// so that zone.Mark/Scan/Get work inside tests.
func TestMain(m *testing.M) {
	zone.NewGlobal()
	os.Exit(m.Run())
}

// renderAndSync calls View (which scans and registers zones) and gives the
// zone manager's async worker time to record the marked regions before the
// caller reads them back with zone.Get.
func renderAndSync(m Model) {
	_ = m.View()
	// zone.Scan buffers zone info to a worker goroutine; wait for it to drain.
	time.Sleep(25 * time.Millisecond)
}

func leftClick(z *zone.ZoneInfo) tea.MouseMsg {
	return tea.MouseMsg{
		X:      z.StartX,
		Y:      z.StartY,
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
	}
}

func TestMouseTabClickFocusesBot(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40

	renderAndSync(m)
	z := zone.Get("tab:2")
	require.NotNil(t, z)
	require.False(t, z.IsZero())

	updated, _ := m.Update(leftClick(z))
	um := updated.(Model)
	assert.Equal(t, 2, um.active)
}

func TestMouseTabClickResetsInputMode(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.inputMode = inputBet // simulate a pending input

	renderAndSync(m)
	z := zone.Get("tab:1")
	require.NotNil(t, z)

	updated, _ := m.Update(leftClick(z))
	um := updated.(Model)
	assert.Equal(t, 1, um.active)
	assert.Equal(t, inputNone, um.inputMode)
}

func TestMouseActionChipMatchesNumberKey(t *testing.T) {
	newModel := func() Model {
		m := newTestModel()
		m.width = 120
		m.height = 40
		m.bots[0].gameState = &GameState{
			GameName:     gameTexasHoldEm,
			ValidActions: []ValidAction{{Action: actionBet, Name: "Bet"}},
			MinBet:       50,
			MaxBet:       300,
		}
		return m
	}

	// Pressing '1' opens the bet input.
	keyModel := newModel()
	updated, _ := keyModel.Update(keyRune('1'))
	assert.Equal(t, inputBet, updated.(Model).inputMode)

	// Clicking the first action chip must do exactly the same.
	clickModel := newModel()
	renderAndSync(clickModel)
	z := zone.Get("action:1")
	require.NotNil(t, z)

	updated, _ = clickModel.Update(leftClick(z))
	assert.Equal(t, inputBet, updated.(Model).inputMode)
}

func TestMouseCardClickTogglesSelection(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	hand := []CardInfo{
		{Rank: 14, Suit: "spades"},
		{Rank: 13, Suit: "hearts"},
		{Rank: 12, Suit: "clubs"},
	}
	m.cardSelect = NewCardSelect(ValidAction{Action: actionDiscard, Name: "Discard"}, hand, "Select cards")
	m.inputMode = inputCardSelect

	renderAndSync(m)
	z := zone.Get("card:1")
	require.NotNil(t, z)

	// First click selects card 1 and moves the cursor there.
	updated, _ := m.Update(leftClick(z))
	um := updated.(Model)
	assert.True(t, um.cardSelect.Selected[1])
	assert.Equal(t, 1, um.cardSelect.Cursor)

	// Second click on the same card toggles it back off.
	renderAndSync(um)
	z = zone.Get("card:1")
	require.NotNil(t, z)
	updated, _ = um.Update(leftClick(z))
	um = updated.(Model)
	assert.False(t, um.cardSelect.Selected[1])
}

func TestMouseWheelScrollsLog(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	for range 30 {
		m.logBuf.Add("entry")
	}

	renderAndSync(m)
	z := zone.Get("logpanel")
	require.NotNil(t, z)
	require.False(t, z.IsZero())

	wheel := func(button tea.MouseButton) tea.MouseMsg {
		return tea.MouseMsg{X: z.StartX, Y: z.StartY, Button: button}
	}

	// Wheel up scrolls back by logWheelStep.
	updated, _ := m.Update(wheel(tea.MouseButtonWheelUp))
	um := updated.(Model)
	assert.Equal(t, logWheelStep, um.logScroll)

	// Wheel down returns toward the tail, clamped at zero.
	updated, _ = um.Update(wheel(tea.MouseButtonWheelDown))
	um = updated.(Model)
	assert.Equal(t, 0, um.logScroll)

	updated, _ = um.Update(wheel(tea.MouseButtonWheelDown))
	um = updated.(Model)
	assert.Equal(t, 0, um.logScroll)
}

func TestMouseWheelOutsideLogIgnored(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	for range 30 {
		m.logBuf.Add("entry")
	}

	renderAndSync(m)

	// (0,0) is the header bar, outside the log panel.
	updated, _ := m.Update(tea.MouseMsg{X: 0, Y: 0, Button: tea.MouseButtonWheelUp})
	um := updated.(Model)
	assert.Equal(t, 0, um.logScroll)
}

func TestMouseClickDismissesHelp(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.showHelp = true

	renderAndSync(m)

	// Any left click dismisses help, regardless of position.
	updated, _ := m.Update(tea.MouseMsg{X: 0, Y: 0, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	um := updated.(Model)
	assert.False(t, um.showHelp)
}

func TestMouseOverlayItemClick(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.bots[0].sendCh = make(chan outgoingMessage, 64)
	m.bots[0].closed = make(chan struct{})
	m.overlay = NewOverlay(m.bots)

	renderAndSync(m)
	// Item 1 is "Terminate Game".
	z := zone.Get("overlay:1")
	require.NotNil(t, z)

	updated, _ := m.Update(leftClick(z))
	um := updated.(Model)
	assert.False(t, um.overlay.Active)
	msg := <-um.bots[0].sendCh
	assert.Equal(t, "terminateGame", msg.Action)
}

func TestMouseGameSelectClick(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.bots[0].sendCh = make(chan outgoingMessage, 64)
	m.bots[0].closed = make(chan struct{})
	m.gameSelect = NewGameSelect()

	renderAndSync(m)
	z := zone.Get("gameselect:0")
	require.NotNil(t, z)

	updated, _ := m.Update(leftClick(z))
	um := updated.(Model)
	assert.False(t, um.gameSelect.Active)
	assert.Equal(t, m.gameSelect.Items[0].Name, um.lastGame)
	msg := <-um.bots[0].sendCh
	assert.Equal(t, "createGame", msg.Action)
}

func TestMouseIgnoresNonLeftAndMotion(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	renderAndSync(m)

	z := zone.Get("tab:2")
	require.NotNil(t, z)

	// Motion over a tab must not focus it.
	motion := tea.MouseMsg{X: z.StartX, Y: z.StartY, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft}
	updated, _ := m.Update(motion)
	assert.Equal(t, 0, updated.(Model).active)

	// Right-button release must not focus it either.
	right := tea.MouseMsg{X: z.StartX, Y: z.StartY, Action: tea.MouseActionRelease, Button: tea.MouseButtonRight}
	updated, _ = m.Update(right)
	assert.Equal(t, 0, updated.(Model).active)
}

// TestScannedViewWidthAligned verifies that the bubblezone markers do not
// corrupt the width math: after zone.Scan strips them, no rendered line exceeds
// the terminal width and the full-width header exactly fills it.
func TestScannedViewWidthAligned(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.bots[0].gameState = &GameState{
		GameName:     gameTexasHoldEm,
		ValidActions: []ValidAction{{Action: "check", Name: "Check"}, {Action: actionBet, Name: "Bet"}},
		MinBet:       50,
		MaxBet:       300,
	}

	view := m.View() // scanned output, markers stripped
	lines := strings.Split(view, "\n")
	require.NotEmpty(t, lines)

	// Header bar (line 0) is padded to the full width.
	assert.Equal(t, m.width, lipgloss.Width(lines[0]))

	for i, ln := range lines {
		assert.LessOrEqualf(t, lipgloss.Width(ln), m.width, "line %d exceeds width: %q", i, ln)
	}
}
