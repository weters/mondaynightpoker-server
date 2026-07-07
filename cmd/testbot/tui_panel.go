package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// panel wraps content in a rounded border with a title embedded in the top
// border, e.g.:
//
//	╭─ Title ─────────╮
//	│ content         │
//	╰─────────────────╯
//
// width and height are the total outer dimensions including the border.
// focused panels get a highlighted border.
func panel(title, content string, width, height int, focused bool) string {
	if width < 8 {
		width = 8
	}
	if height < 3 {
		height = 3
	}

	borderColor := colorComment
	if focused {
		borderColor = colorPurple
	}
	border := lipgloss.NewStyle().Foreground(borderColor)

	// Body: left/right/bottom borders; the top border is built by hand so
	// the title can sit inside it.
	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, true, true, true).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width - 2).
		Height(height - 2).
		Render(content)

	fill := width - lipgloss.Width(title) - 5
	if fill < 0 {
		fill = 0
	}
	top := border.Render("╭─ ") +
		stylePanelTitle.Render(title) +
		border.Render(" "+strings.Repeat("─", fill)+"╮")

	return top + "\n" + body
}
