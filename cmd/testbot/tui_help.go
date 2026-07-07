package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// helpBinding is a single key/description pair in the help overlay.
type helpBinding struct {
	Key  string
	Desc string
}

var helpBindings = []helpBinding{
	{"1-9", "perform the numbered action for the focused bot"},
	{"tab / shift+tab", "focus next / previous bot"},
	{"d", "toggle dashboard (all bots at once)"},
	{"a", "toggle auto-pilot for the focused bot"},
	{"A", "toggle auto-pilot for all bots"},
	{"s", "cycle auto-pilot speed (normal/fast/instant/slow)"},
	{"g", "start a game (game picker)"},
	{"r", "restart the last game"},
	{"T", "terminate the current game"},
	{"pgup / pgdn", "scroll the game log"},
	{"end", "jump log back to live tail"},
	{"esc", "open menu"},
	{"?", "toggle this help"},
	{"ctrl+c", "quit"},
}

// RenderHelp renders the help overlay centered on screen.
func RenderHelp(width, height int) string {
	title := styleOverlayTitle.Render(" Keys ")

	keyWidth := 0
	for _, b := range helpBindings {
		if len(b.Key) > keyWidth {
			keyWidth = len(b.Key)
		}
	}

	lines := make([]string, len(helpBindings))
	for i, b := range helpBindings {
		key := b.Key + strings.Repeat(" ", keyWidth-len(b.Key))
		lines[i] = styleFooterKey.Render(key) + "  " + styleOverlayItem.Render(b.Desc)
	}

	hint := styleOverlayHint.Render("press any key to close")
	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" + hint
	box := styleOverlayBorder.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
