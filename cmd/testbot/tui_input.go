package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// inputMode tracks which sub-input is currently active.
type inputMode int

const (
	inputNone inputMode = iota
	inputBet
	inputCardSelect
)

// BetInputModel handles numeric bet amount entry.
type BetInputModel struct {
	Active  bool
	Value   string
	MinBet  int
	MaxBet  int
	Action  ValidAction
	ErrText string
}

// NewBetInput creates a bet input model for the given range.
func NewBetInput(action ValidAction, minBet, maxBet int) BetInputModel {
	return BetInputModel{
		Active: true,
		MinBet: minBet,
		MaxBet: maxBet,
		Action: action,
	}
}

// Update processes a key press for the bet input.
// Returns the model, an optional completed amount (or -1), and whether to cancel.
func (m BetInputModel) Update(msg tea.KeyMsg) (BetInputModel, int, bool) {
	switch msg.Type {
	case tea.KeyEscape:
		return m, -1, true
	case tea.KeyEnter:
		if m.Value == "" {
			return m, m.MinBet, false
		}
		amt, err := strconv.Atoi(m.Value)
		if err != nil {
			m.ErrText = "enter a number"
			return m, -1, false
		}
		if amt < m.MinBet || amt > m.MaxBet {
			m.ErrText = fmt.Sprintf("must be %d-%d", m.MinBet, m.MaxBet)
			return m, -1, false
		}
		if amt%betIncrement != 0 {
			m.ErrText = fmt.Sprintf("must be in increments of $%d", betIncrement)
			return m, -1, false
		}
		return m, amt, false
	case tea.KeyBackspace:
		if len(m.Value) > 0 {
			m.Value = m.Value[:len(m.Value)-1]
			m.ErrText = ""
		}
		return m, -1, false
	default:
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if r >= '0' && r <= '9' {
					m.Value += string(r)
					m.ErrText = ""
				}
			}
		}
		return m, -1, false
	}
}

// View renders the bet input inline.
func (m BetInputModel) View() string {
	prompt := styleInputLabel.Render(fmt.Sprintf("Bet amount (%d-%d): ", m.MinBet, m.MaxBet))
	val := styleInputValue.Render(m.Value + "█")
	line := prompt + val
	if m.ErrText != "" {
		line += " " + styleError.Render(m.ErrText)
	}
	return line
}

// CardSelectModel handles multi-select card picking.
type CardSelectModel struct {
	Active   bool
	Cards    []CardInfo
	Selected []bool
	Cursor   int
	Title    string
	Action   ValidAction
}

// NewCardSelect creates a card selection model.
func NewCardSelect(action ValidAction, cards []CardInfo, title string) CardSelectModel {
	return CardSelectModel{
		Active:   true,
		Cards:    cards,
		Selected: make([]bool, len(cards)),
		Title:    title,
		Action:   action,
	}
}

// Update processes key input for card selection.
// Returns the model, selected cards (nil if not done), and cancel flag.
func (m CardSelectModel) Update(msg tea.KeyMsg) (CardSelectModel, []CardInfo, bool) {
	switch msg.Type {
	case tea.KeyEscape:
		return m, nil, true
	case tea.KeyEnter:
		selected := make([]CardInfo, 0)
		for i, s := range m.Selected {
			if s {
				selected = append(selected, m.Cards[i])
			}
		}
		return m, selected, false
	case tea.KeyLeft:
		if m.Cursor > 0 {
			m.Cursor--
		}
		return m, nil, false
	case tea.KeyRight:
		if m.Cursor < len(m.Cards)-1 {
			m.Cursor++
		}
		return m, nil, false
	case tea.KeySpace:
		if m.Cursor >= 0 && m.Cursor < len(m.Cards) {
			m.Selected[m.Cursor] = !m.Selected[m.Cursor]
		}
		return m, nil, false
	default:
		return m, nil, false
	}
}

// View renders the card selection as art cards with cursor/selection states.
func (m CardSelectModel) View(width int) string {
	title := styleInputLabel.Render(m.Title + " (←/→ move, space select, enter confirm, esc cancel)")

	// Fall back to inline text when the small art cards would not fit.
	if !m.artFits(width) {
		return title + "\n" + m.inlineView()
	}

	blocks := make([]string, len(m.Cards))
	for i, c := range m.Cards {
		blocks[i] = zone.Mark(fmt.Sprintf("card:%d", i), m.renderCardBlock(i, c))
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, intersperse(blocks, " ")...)
	return title + "\n" + body
}

// artFits reports whether the small art cards fit within width.
func (m CardSelectModel) artFits(width int) bool {
	return width > 0 && handArtWidth(len(m.Cards), smallCardWidth) <= width
}

// renderCardBlock renders one card plus a marker row, raising selected cards.
func (m CardSelectModel) renderCardBlock(i int, c CardInfo) string {
	border := cardStyle(c.Suit)
	switch {
	case i == m.Cursor:
		border = styleCardCursor
	case m.Selected[i]:
		border = styleCardSelected
	}
	card := renderSmallCard(c, border)

	blank := strings.Repeat(" ", smallCardWidth)
	if m.Selected[i] {
		// Raise the card one line and mark it as selected below.
		mark := styleCardMark.Render(centerTo("✔", smallCardWidth))
		return card + "\n" + mark
	}
	// Push unselected cards down one line so selected cards sit higher.
	return blank + "\n" + card
}

// inlineView renders the narrow-width fallback with cursor/selection markers.
func (m CardSelectModel) inlineView() string {
	parts := make([]string, len(m.Cards))
	for i, c := range m.Cards {
		style := cardStyle(c.Suit)
		label := c.String()
		if m.Selected[i] {
			label = "[" + label + "]"
			style = styleCardSelected
		} else {
			label = " " + label + " "
		}
		if i == m.Cursor {
			label = ">" + label + "<"
			style = styleCardCursor
		}
		parts[i] = zone.Mark(fmt.Sprintf("card:%d", i), style.Render(label))
	}
	return strings.Join(parts, " ")
}
