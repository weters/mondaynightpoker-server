package main

import "github.com/charmbracelet/lipgloss"

// Dracula-inspired color palette.
const (
	colorBackground = lipgloss.Color("#282A36")
	colorForeground = lipgloss.Color("#F8F8F2")
	colorComment    = lipgloss.Color("#6272A4")
	colorCyan       = lipgloss.Color("#8BE9FD")
	colorGreen      = lipgloss.Color("#50FA7B")
	colorOrange     = lipgloss.Color("#FFB86C")
	colorPink       = lipgloss.Color("#FF79C6")
	colorPurple     = lipgloss.Color("#BD93F9")
	colorRed        = lipgloss.Color("#FF5555")
	colorYellow     = lipgloss.Color("#F1FA8C")
)

var (
	// styleBorder is the main outer border for the TUI.
	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorComment)

	// styleHeader renders the top bar (player name, game name, pot).
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan)

	// stylePot renders the pot amount.
	stylePot = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGreen)

	// styleCardRed renders red suit cards (hearts, diamonds).
	styleCardRed = lipgloss.NewStyle().
			Foreground(colorRed)

	// styleCardWhite renders black suit cards (clubs, spades).
	styleCardWhite = lipgloss.NewStyle().
			Foreground(colorForeground)

	// styleAction renders action keys in the action bar.
	styleAction = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow)

	// styleActionLabel renders action descriptions.
	styleActionLabel = lipgloss.NewStyle().
				Foreground(colorForeground)

	// styleLogDivider renders the log section divider.
	styleLogDivider = lipgloss.NewStyle().
			Foreground(colorComment)

	// styleLogEntry renders individual log entries.
	styleLogEntry = lipgloss.NewStyle().
			Foreground(colorComment)

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

	// styleOverlayBorder renders the overlay menu border.
	styleOverlayBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPurple).
				Padding(1, 2)

	// styleOverlayTitle renders the overlay title.
	styleOverlayTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPurple)

	// styleOverlaySelected renders the selected overlay item.
	styleOverlaySelected = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPink)

	// styleOverlayItem renders unselected overlay items.
	styleOverlayItem = lipgloss.NewStyle().
				Foreground(colorForeground)

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

	// styleSectionLabel renders section labels like "Your Hand:", "Community:".
	styleSectionLabel = lipgloss.NewStyle().
				Foreground(colorComment)
)
