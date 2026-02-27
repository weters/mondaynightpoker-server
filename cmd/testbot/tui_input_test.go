package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestBetInputEscape(t *testing.T) {
	m := NewBetInput(ValidAction{Action: "bet", Name: "Bet"}, 50, 300)
	assert.True(t, m.Active)

	_, amt, cancel := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.True(t, cancel)
	assert.Equal(t, -1, amt)
}

func TestBetInputEnterEmpty(t *testing.T) {
	m := NewBetInput(ValidAction{Action: "bet", Name: "Bet"}, 50, 300)

	_, amt, cancel := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, cancel)
	assert.Equal(t, 50, amt) // defaults to min
}

func TestBetInputEnterValue(t *testing.T) {
	m := NewBetInput(ValidAction{Action: "bet", Name: "Bet"}, 50, 300)

	// Type "150"
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})

	assert.Equal(t, "150", m.Value)

	_, amt, cancel := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, cancel)
	assert.Equal(t, 150, amt)
}

func TestBetInputOutOfRange(t *testing.T) {
	m := NewBetInput(ValidAction{Action: "bet", Name: "Bet"}, 50, 300)

	// Type "999"
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})

	m, amt, cancel := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, cancel)
	assert.Equal(t, -1, amt) // rejected
	assert.NotEmpty(t, m.ErrText)
}

func TestBetInputBackspace(t *testing.T) {
	m := NewBetInput(ValidAction{Action: "bet", Name: "Bet"}, 50, 300)

	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	assert.Equal(t, "15", m.Value)

	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "1", m.Value)
}

func TestBetInputView(t *testing.T) {
	m := NewBetInput(ValidAction{Action: "bet", Name: "Bet"}, 50, 300)
	view := m.View()
	assert.Contains(t, view, "50-300")
}

func TestCardSelectBasic(t *testing.T) {
	cards := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "spades"},
		{Rank: 12, Suit: "diamonds"},
	}

	m := NewCardSelect(ValidAction{Action: "discard", Name: "Discard"}, cards, "Select cards")
	assert.True(t, m.Active)
	assert.Len(t, m.Selected, 3)
	assert.Equal(t, 0, m.Cursor)
}

func TestCardSelectNavigation(t *testing.T) {
	cards := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "spades"},
	}

	m := NewCardSelect(ValidAction{Action: "discard", Name: "Discard"}, cards, "Select")

	// Move right
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.Cursor)

	// Move right again (should stay at 1)
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.Cursor)

	// Move left
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 0, m.Cursor)
}

func TestCardSelectToggle(t *testing.T) {
	cards := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "spades"},
	}

	m := NewCardSelect(ValidAction{Action: "discard", Name: "Discard"}, cards, "Select")

	// Select first card
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.True(t, m.Selected[0])
	assert.False(t, m.Selected[1])

	// Deselect
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.False(t, m.Selected[0])
}

func TestCardSelectConfirm(t *testing.T) {
	cards := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "spades"},
	}

	m := NewCardSelect(ValidAction{Action: "discard", Name: "Discard"}, cards, "Select")

	// Select first card
	m, _, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})

	// Confirm
	_, selected, cancel := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, cancel)
	assert.Len(t, selected, 1)
	assert.Equal(t, cards[0], selected[0])
}

func TestCardSelectEscape(t *testing.T) {
	cards := []CardInfo{{Rank: 14, Suit: "hearts"}}
	m := NewCardSelect(ValidAction{Action: "discard", Name: "Discard"}, cards, "Select")

	_, _, cancel := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.True(t, cancel)
}

func TestCardSelectConfirmNoSelection(t *testing.T) {
	cards := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "spades"},
	}

	m := NewCardSelect(ValidAction{Action: "trade", Name: "Trade Cards"}, cards, "Select cards to trade")

	// Confirm without selecting any cards (stand pat)
	_, selected, cancel := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, cancel)
	assert.NotNil(t, selected, "selected should be non-nil empty slice, not nil")
	assert.Len(t, selected, 0)
}

func TestCardSelectView(t *testing.T) {
	cards := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "spades"},
	}
	m := NewCardSelect(ValidAction{Action: "discard", Name: "Discard"}, cards, "Select cards")
	view := m.View(80)
	assert.Contains(t, view, "Select cards")
}
