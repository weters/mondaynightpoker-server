package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// minWidthForArt is the minimum terminal width to show ASCII card art.
const minWidthForArt = 60

// suitSymbol returns the unicode suit symbol.
func suitSymbol(suit string) string {
	switch suit {
	case "hearts":
		return "♥"
	case "diamonds":
		return "♦"
	case "clubs":
		return "♣"
	case "spades":
		return "♠"
	default:
		return suit
	}
}

// isRedSuit returns true for hearts and diamonds.
func isRedSuit(suit string) bool {
	return suit == "hearts" || suit == "diamonds"
}

// cardStyle returns the appropriate style for a card's suit.
func cardStyle(suit string) lipgloss.Style {
	if isRedSuit(suit) {
		return styleCardRed
	}
	return styleCardWhite
}

// RenderCard renders a single card as a 6×4 ASCII art block.
//
//	┌────┐
//	│ A  │
//	│ ♥  │
//	└────┘
func RenderCard(c CardInfo) string {
	r := cardRankDisplay(c.Rank)
	s := suitSymbol(c.Suit)
	style := cardStyle(c.Suit)

	// Pad rank to 2 chars for alignment
	rankPad := r
	if len(r) == 1 {
		rankPad = r + " "
	}

	top := style.Render("┌────┐")
	mid1 := style.Render("│ " + rankPad + " │")
	mid2 := style.Render("│ " + s + "  │")
	bot := style.Render("└────┘")

	return top + "\n" + mid1 + "\n" + mid2 + "\n" + bot
}

// cardRankDisplay returns the display string for rendering inside card art.
func cardRankDisplay(rank int) string {
	ranks := map[int]string{
		14: "A", 13: "K", 12: "Q", 11: "J", 10: "10",
		9: "9", 8: "8", 7: "7", 6: "6", 5: "5",
		4: "4", 3: "3", 2: "2",
	}
	if r, ok := ranks[rank]; ok {
		return r
	}
	return "?"
}

// RenderHand renders multiple cards side by side as ASCII art.
// Falls back to inline text (e.g., "A♥ K♠") if width < minWidthForArt.
func RenderHand(cards []CardInfo, availableWidth int) string {
	if len(cards) == 0 {
		return ""
	}

	if availableWidth < minWidthForArt {
		return RenderHandInline(cards)
	}

	return RenderHandArt(cards)
}

// RenderHandInline renders cards as inline text: "A♥ K♠ Q♦"
func RenderHandInline(cards []CardInfo) string {
	parts := make([]string, len(cards))
	for i, c := range cards {
		style := cardStyle(c.Suit)
		parts[i] = style.Render(c.String())
	}
	return strings.Join(parts, " ")
}

// RenderHandArt renders cards side by side as ASCII art blocks.
func RenderHandArt(cards []CardInfo) string {
	if len(cards) == 0 {
		return ""
	}

	// Render each card individually
	rendered := make([]string, len(cards))
	for i, c := range cards {
		rendered[i] = RenderCard(c)
	}

	// Join cards horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, intersperse(rendered, " ")...)
}

// intersperse inserts sep between each element.
func intersperse(items []string, sep string) []string {
	if len(items) <= 1 {
		return items
	}
	result := make([]string, 0, len(items)*2-1)
	for i, item := range items {
		if i > 0 {
			result = append(result, sep)
		}
		result = append(result, item)
	}
	return result
}
