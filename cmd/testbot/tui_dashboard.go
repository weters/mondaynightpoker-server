package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"mondaynightpoker-server/pkg/money"
)

// RenderDashboard renders a one-line-per-bot overview: focus marker, name,
// status, balance, and hand. It gives a whole-table view for testing
// showdowns and multi-player flows.
func RenderDashboard(bots []*Bot, active int) string {
	nameWidth := 0
	for _, b := range bots {
		label := fmt.Sprintf("p%d %s", b.ID, b.Name)
		if len(label) > nameWidth {
			nameWidth = len(label)
		}
	}

	lines := make([]string, 0, len(bots))

	for i, b := range bots {
		gs := b.GetGameState()

		marker := "  "
		if i == active {
			marker = styleOverlaySelected.Render("▸ ")
		}

		label := fmt.Sprintf("p%d %s", b.ID, b.Name)
		label += strings.Repeat(" ", nameWidth-len(label))

		status, style := botStatus(b, gs)
		statusCol := style.Render(fmt.Sprintf("%-5s", status))

		balance := fmt.Sprintf("%8s", "-")
		if gs != nil && gs.Balance != 0 {
			balance = fmt.Sprintf("%8s", money.FormatCents(gs.Balance))
		}

		hand := ""
		if gs != nil && len(gs.Hand) > 0 {
			hand = RenderHandInline(gs.Hand)
		}

		line := marker + styleHeader.Render(label) + "  " + statusCol + "  " + stylePot.Render(balance) + "  " + hand
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// botStatus returns a short status string and its display style.
func botStatus(b *Bot, gs *GameState) (string, lipgloss.Style) {
	switch {
	case b.Disconnected():
		return "off", styleError
	case b.AutoPilot:
		return "auto", styleStatusAuto
	case gs != nil && len(gs.ValidActions) > 0:
		return "ACT", styleStatusActive
	default:
		return "idle", styleStatusIdle
	}
}
