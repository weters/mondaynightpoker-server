package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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

// View renders the card selection inline.
func (m CardSelectModel) View(width int) string {
	title := styleInputLabel.Render(m.Title + " (←/→ move, space select, enter confirm, esc cancel)")

	parts := make([]string, len(m.Cards))
	for i, c := range m.Cards {
		style := cardStyle(c.Suit)
		label := c.String()
		if m.Selected[i] {
			label = "[" + label + "]"
		} else {
			label = " " + label + " "
		}
		if i == m.Cursor {
			label = ">" + label + "<"
			style = style.Bold(true)
		}
		parts[i] = style.Render(label)
	}

	_ = width // reserved for future use
	return title + "\n" + strings.Join(parts, " ")
}
