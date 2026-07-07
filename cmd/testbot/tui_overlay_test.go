package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewOverlay(t *testing.T) {
	bots := []*Bot{
		{ID: 1, Name: "Alice", AutoPilot: false},
		{ID: 2, Name: "Bob", AutoPilot: true},
	}

	m := NewOverlay(bots)
	assert.True(t, m.Active)
	// Should have: start, terminate, cancel-pending, toggle:1, toggle:2, toggle-all, quit = 7 items
	assert.Len(t, m.Items, 7)
	assert.Equal(t, "start", m.Items[0].Action)
	assert.Equal(t, "terminate", m.Items[1].Action)
	assert.Equal(t, "cancel-pending", m.Items[2].Action)
	assert.Equal(t, "toggle:1", m.Items[3].Action)
	assert.Equal(t, "toggle:2", m.Items[4].Action)
	assert.Equal(t, "toggle-all", m.Items[5].Action)
	assert.Equal(t, "quit", m.Items[6].Action)
}

func TestOverlayNavigation(t *testing.T) {
	bots := []*Bot{{ID: 1, Name: "Alice"}}
	m := NewOverlay(bots)

	// Move down
	m, action := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, "", action)
	assert.Equal(t, 1, m.Cursor)

	// Move up
	m, action = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, "", action)
	assert.Equal(t, 0, m.Cursor)

	// Move up again (stay at 0)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.Cursor)
}

func TestOverlayEscape(t *testing.T) {
	bots := []*Bot{{ID: 1, Name: "Alice"}}
	m := NewOverlay(bots)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, m.Active)
}

func TestOverlaySelect(t *testing.T) {
	bots := []*Bot{{ID: 1, Name: "Alice"}}
	m := NewOverlay(bots)

	// First item is "start"
	_, action := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "start", action)
}

func TestOverlayAutoPilotLabels(t *testing.T) {
	bots := []*Bot{
		{ID: 1, Name: "Alice", AutoPilot: false},
		{ID: 2, Name: "Bob", AutoPilot: true},
	}

	m := NewOverlay(bots)
	assert.Contains(t, m.Items[3].Label, "[OFF]")
	assert.Contains(t, m.Items[4].Label, "[ON]")
}

func TestOverlayView(t *testing.T) {
	bots := []*Bot{{ID: 1, Name: "Alice"}}
	m := NewOverlay(bots)
	view := m.View(80, 24)
	assert.Contains(t, view, "Menu")
	assert.Contains(t, view, "Start Game")
}

func TestGameSelectBasic(t *testing.T) {
	m := NewGameSelect()
	assert.True(t, m.Active)
	assert.Greater(t, len(m.Items), 0)
}

func TestGameSelectNavAndConfirm(t *testing.T) {
	m := NewGameSelect()

	// Move to second item
	m, gameName := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, "", gameName)
	assert.Equal(t, 1, m.Cursor)

	// Select it
	_, gameName = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, m.Items[1].Name, gameName)
}

func TestGameSelectEscape(t *testing.T) {
	m := NewGameSelect()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, m.Active)
}

func TestGameSelectView(t *testing.T) {
	m := NewGameSelect()
	view := m.View(80, 24)
	assert.Contains(t, view, "Select a Game")
	assert.Contains(t, view, "Texas Hold'em")
}
