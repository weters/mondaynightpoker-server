package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// overlayItem represents a menu item in the ESC overlay.
type overlayItem struct {
	Label  string
	Action string
}

// OverlayModel is the ESC menu overlay.
type OverlayModel struct {
	Active bool
	Cursor int
	Items  []overlayItem
}

// NewOverlay creates the overlay menu with items based on current bot state.
func NewOverlay(bots []*Bot) OverlayModel {
	items := []overlayItem{
		{Label: "Start Game", Action: "start"},
	}

	// Add per-bot autopilot toggles
	for _, b := range bots {
		label := fmt.Sprintf("p%d %s: auto-pilot ", b.ID, b.Name)
		if b.AutoPilot {
			label += "[ON]"
		} else {
			label += "[OFF]"
		}
		items = append(items, overlayItem{
			Label:  label,
			Action: fmt.Sprintf("toggle:%d", b.ID),
		})
	}

	// Toggle all
	allAuto := true
	for _, b := range bots {
		if !b.AutoPilot {
			allAuto = false
			break
		}
	}
	if allAuto {
		items = append(items, overlayItem{Label: "All auto-pilot [ON → OFF]", Action: "toggle-all"})
	} else {
		items = append(items, overlayItem{Label: "All auto-pilot [OFF → ON]", Action: "toggle-all"})
	}

	items = append(items, overlayItem{Label: "Quit", Action: "quit"})

	return OverlayModel{
		Active: true,
		Items:  items,
	}
}

// Update handles key events in the overlay.
// Returns the model and the selected action (empty string if none).
func (m OverlayModel) Update(msg tea.KeyMsg) (OverlayModel, string) {
	switch msg.Type {
	case tea.KeyEscape:
		m.Active = false
		return m, ""
	case tea.KeyUp, tea.KeyShiftTab:
		if m.Cursor > 0 {
			m.Cursor--
		}
		return m, ""
	case tea.KeyDown, tea.KeyTab:
		if m.Cursor < len(m.Items)-1 {
			m.Cursor++
		}
		return m, ""
	case tea.KeyEnter:
		if m.Cursor >= 0 && m.Cursor < len(m.Items) {
			return m, m.Items[m.Cursor].Action
		}
		return m, ""
	default:
		return m, ""
	}
}

// View renders the overlay menu centered on screen.
func (m OverlayModel) View(width, height int) string {
	title := styleOverlayTitle.Render("  Menu  ")

	lines := make([]string, len(m.Items))
	for i, item := range m.Items {
		if i == m.Cursor {
			lines[i] = styleOverlaySelected.Render("▸ " + item.Label)
		} else {
			lines[i] = styleOverlayItem.Render("  " + item.Label)
		}
	}

	content := title + "\n\n" + strings.Join(lines, "\n")
	box := styleOverlayBorder.Render(content)

	// Center the overlay
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// gameMenuItem represents a game choice in the start-game sub-menu.
type gameMenuItem struct {
	Label string
	Name  string
}

// gameMenuItems is the list of available games.
var gameMenuItems = []gameMenuItem{
	{Label: "Texas Hold'em", Name: gameTexasHoldEm},
	{Label: "Texas Hold'em (PLO)", Name: gameTexasHoldEmPLO},
	{Label: "Bourre", Name: gameBourre},
	{Label: "Guts", Name: gameGuts},
	{Label: "Pass the Poop", Name: gamePassThePoop},
	{Label: "Acey Deucey", Name: gameAceyDeucey},
	{Label: "Seven Card", Name: gameSevenCard},
	{Label: "Little L", Name: gameLittleL},
}

// GameSelectModel is the game selection sub-menu.
type GameSelectModel struct {
	Active bool
	Cursor int
	Items  []gameMenuItem
}

// NewGameSelect creates a game selection menu.
func NewGameSelect() GameSelectModel {
	return GameSelectModel{
		Active: true,
		Items:  gameMenuItems,
	}
}

// Update handles key events for game selection.
// Returns the model and the selected game name (empty if none).
func (m GameSelectModel) Update(msg tea.KeyMsg) (GameSelectModel, string) {
	switch msg.Type {
	case tea.KeyEscape:
		m.Active = false
		return m, ""
	case tea.KeyUp, tea.KeyShiftTab:
		if m.Cursor > 0 {
			m.Cursor--
		}
		return m, ""
	case tea.KeyDown, tea.KeyTab:
		if m.Cursor < len(m.Items)-1 {
			m.Cursor++
		}
		return m, ""
	case tea.KeyEnter:
		if m.Cursor >= 0 && m.Cursor < len(m.Items) {
			return m, m.Items[m.Cursor].Name
		}
		return m, ""
	default:
		return m, ""
	}
}

// View renders the game selection menu.
func (m GameSelectModel) View(width, height int) string {
	title := styleOverlayTitle.Render("  Select a Game  ")

	lines := make([]string, len(m.Items))
	for i, item := range m.Items {
		if i == m.Cursor {
			lines[i] = styleOverlaySelected.Render("▸ " + item.Label)
		} else {
			lines[i] = styleOverlayItem.Render("  " + item.Label)
		}
	}

	content := title + "\n\n" + strings.Join(lines, "\n")
	box := styleOverlayBorder.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
