package main

import "github.com/charmbracelet/lipgloss"

// Dracula-inspired color palette.
const (
	colorBackground  = lipgloss.Color("#282A36")
	colorCurrentLine = lipgloss.Color("#44475A")
	colorForeground  = lipgloss.Color("#F8F8F2")
	colorComment     = lipgloss.Color("#6272A4")
	colorCyan        = lipgloss.Color("#8BE9FD")
	colorGreen       = lipgloss.Color("#50FA7B")
	colorOrange      = lipgloss.Color("#FFB86C")
	colorPink        = lipgloss.Color("#FF79C6")
	colorPurple      = lipgloss.Color("#BD93F9")
	colorRed         = lipgloss.Color("#FF5555")
	colorYellow      = lipgloss.Color("#F1FA8C")
)

var (
	// styleAppHeader renders the full-width top title bar.
	styleAppHeader = lipgloss.NewStyle().
			Background(colorCurrentLine).
			Foreground(colorForeground).
			Bold(true)

	// styleTabActive renders the focused bot tab.
	styleTabActive = lipgloss.NewStyle().
			Background(colorPurple).
			Foreground(colorBackground).
			Bold(true).
			Padding(0, 1)

	// styleTabAlert renders a bot tab that has pending actions.
	styleTabAlert = lipgloss.NewStyle().
			Background(colorGreen).
			Foreground(colorBackground).
			Bold(true).
			Padding(0, 1)

	// styleTabInactive renders unfocused bot tabs.
	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorComment).
				Padding(0, 1)

	// stylePanelTitle renders the title embedded in a panel's top border.
	stylePanelTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple)

	// styleHeader renders emphasized labels (player names).
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan)

	// stylePot renders money amounts.
	stylePot = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGreen)

	// styleCardRed renders red suit cards (hearts, diamonds).
	styleCardRed = lipgloss.NewStyle().
			Foreground(colorRed)

	// styleCardWhite renders black suit cards (clubs, spades).
	styleCardWhite = lipgloss.NewStyle().
			Foreground(colorForeground)

	// styleCardYellow renders star suit cards.
	styleCardYellow = lipgloss.NewStyle().
			Foreground(colorYellow)

	// styleCardBack renders the pattern and border of a face-down card.
	styleCardBack = lipgloss.NewStyle().
			Foreground(colorComment)

	// styleCardCursor renders the border of the card under the selection cursor.
	styleCardCursor = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPink)

	// styleCardSelected renders the border of a selected card.
	styleCardSelected = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorGreen)

	// styleCardMark renders the ✔ marker shown under selected cards.
	styleCardMark = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGreen)

	// styleAction renders action keys in the action bar.
	styleAction = lipgloss.NewStyle().
			Bold(true).
			Background(colorCurrentLine).
			Foreground(colorYellow).
			Padding(0, 1)

	// styleActionLabel renders action descriptions.
	styleActionLabel = lipgloss.NewStyle().
				Foreground(colorForeground)

	// styleLogEntry renders individual log entries.
	styleLogEntry = lipgloss.NewStyle().
			Foreground(colorComment)

	// styleLogTime renders log entry timestamps.
	styleLogTime = lipgloss.NewStyle().
			Foreground(colorCurrentLine)

	// styleLogScroll renders the log scroll indicator.
	styleLogScroll = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow)

	// styleStatusActive renders the active bot indicator.
	styleStatusActive = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorGreen)

	// styleStatusAuto renders autopilot bot indicators.
	styleStatusAuto = lipgloss.NewStyle().
			Foreground(colorOrange)

	// styleStatusIdle renders idle bot indicators.
	styleStatusIdle = lipgloss.NewStyle().
			Foreground(colorComment)

	// styleOverlayBorder renders modal dialog borders.
	styleOverlayBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPurple).
				Padding(1, 3)

	// styleOverlayTitle renders modal dialog titles.
	styleOverlayTitle = lipgloss.NewStyle().
				Bold(true).
				Background(colorPurple).
				Foreground(colorBackground).
				Padding(0, 1)

	// styleOverlaySelected renders the selected modal item.
	styleOverlaySelected = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPink)

	// styleOverlayItem renders unselected modal items.
	styleOverlayItem = lipgloss.NewStyle().
				Foreground(colorForeground)

	// styleOverlayHint renders the key hints at the bottom of modals.
	styleOverlayHint = lipgloss.NewStyle().
				Foreground(colorComment)

	// styleInputLabel renders input prompt labels.
	styleInputLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan)

	// styleInputValue renders the current input value.
	styleInputValue = lipgloss.NewStyle().
			Foreground(colorForeground)

	// styleError renders error messages.
	styleError = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorRed)

	// styleSectionLabel renders section labels like "Your Hand", "Community".
	styleSectionLabel = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorComment)

	// styleFooterHint renders the key hints in the footer bar.
	styleFooterHint = lipgloss.NewStyle().
			Foreground(colorComment)

	// styleFooterKey renders key names in the footer bar.
	styleFooterKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow)
)
