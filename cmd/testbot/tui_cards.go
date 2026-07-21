package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// minWidthForArt is the minimum terminal width to show ASCII card art.
const minWidthForArt = 60

// Card art dimensions (in terminal cells) for the two art tiers.
const (
	smallCardWidth = 6 // 4 interior cells + 2 border cells
	smallInnerCols = 4
	largeCardWidth = 9 // 7 interior cells + 2 border cells
	largeInnerCols = 7
	cardGap        = 1 // horizontal gap between rendered cards
)

const suitStars = "stars"

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
	case suitStars:
		return "⭐"
	default:
		return suit
	}
}

// isRedSuit returns true for hearts and diamonds.
func isRedSuit(suit string) bool {
	return suit == "hearts" || suit == "diamonds"
}

// isHidden reports whether a card represents a face-down / unknown card.
// The zero value (rank 0, empty suit) is treated as hidden.
func isHidden(c CardInfo) bool {
	return c.Rank == 0 || c.Suit == ""
}

// cardStyle returns the appropriate style for a card's suit.
func cardStyle(suit string) lipgloss.Style {
	if isRedSuit(suit) {
		return styleCardRed
	}
	if suit == suitStars {
		return styleCardYellow
	}
	return styleCardWhite
}

// cardFrame holds the box-drawing characters used to frame a card.
type cardFrame struct {
	topLeft, topRight, botLeft, botRight, horiz, vert string
}

var (
	frameSquare = cardFrame{"┌", "┐", "└", "┘", "─", "│"}
	frameRound  = cardFrame{"╭", "╮", "╰", "╯", "─", "│"}
)

// padRightTo pads s on the right with spaces to reach w display cells.
func padRightTo(s string, w int) string {
	if diff := w - lipgloss.Width(s); diff > 0 {
		return s + strings.Repeat(" ", diff)
	}
	return s
}

// padLeftTo pads s on the left with spaces to reach w display cells.
func padLeftTo(s string, w int) string {
	if diff := w - lipgloss.Width(s); diff > 0 {
		return strings.Repeat(" ", diff) + s
	}
	return s
}

// centerTo centers s within w display cells.
func centerTo(s string, w int) string {
	diff := w - lipgloss.Width(s)
	if diff <= 0 {
		return s
	}
	left := diff / 2
	right := diff - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// renderCardBox frames the given interior lines with a border. The border
// characters use borderStyle while the interior content uses contentStyle.
// Each interior line must already be padded to exactly inner display cells so
// that every rendered line has an equal lipgloss.Width of inner+2.
func renderCardBox(f cardFrame, inner int, lines []string, borderStyle, contentStyle lipgloss.Style) string {
	top := borderStyle.Render(f.topLeft + strings.Repeat(f.horiz, inner) + f.topRight)
	bot := borderStyle.Render(f.botLeft + strings.Repeat(f.horiz, inner) + f.botRight)
	v := borderStyle.Render(f.vert)

	out := make([]string, 0, len(lines)+2)
	out = append(out, top)
	for _, ln := range lines {
		out = append(out, v+contentStyle.Render(ln)+v)
	}
	out = append(out, bot)
	return strings.Join(out, "\n")
}

// smallCardLines returns the 2 interior lines for the small (6×4) tier.
func smallCardLines(c CardInfo) []string {
	rank := cardRankDisplay(c.Rank)
	suit := suitSymbol(c.Suit)
	return []string{
		" " + padRightTo(rank, 2) + " ", // rank, top-left
		centerTo(suit, smallInnerCols),  // suit, centered
	}
}

// largeCardLines returns the 5 interior lines for the large (9×7) tier.
func largeCardLines(c CardInfo) []string {
	rank := cardRankDisplay(c.Rank)
	suit := suitSymbol(c.Suit)
	corner := rank + suit
	return []string{
		padRightTo(" "+corner, largeInnerCols), // rank+suit, top-left
		strings.Repeat(" ", largeInnerCols),
		centerTo(suit, largeInnerCols), // centered suit pip
		strings.Repeat(" ", largeInnerCols),
		padLeftTo(corner+" ", largeInnerCols), // mirrored rank+suit, bottom-right
	}
}

// backLines returns interior pattern lines for a face-down card.
func backLines(inner, rows int) []string {
	lines := make([]string, rows)
	for i := range lines {
		lines[i] = strings.Repeat("▒", inner)
	}
	return lines
}

// renderSmallCard renders a card in the small (6×4) tier using borderStyle for
// the frame. Hidden cards render a patterned back.
func renderSmallCard(c CardInfo, borderStyle lipgloss.Style) string {
	if isHidden(c) {
		return renderCardBox(frameSquare, smallInnerCols, backLines(smallInnerCols, 2), styleCardBack, styleCardBack)
	}
	return renderCardBox(frameSquare, smallInnerCols, smallCardLines(c), borderStyle, cardStyle(c.Suit))
}

// RenderCard renders a single card as a 6×4 ASCII art block.
//
//	┌────┐
//	│ A  │
//	│ ♥  │
//	└────┘
func RenderCard(c CardInfo) string {
	if isHidden(c) {
		return renderSmallCard(c, styleCardBack)
	}
	return renderSmallCard(c, cardStyle(c.Suit))
}

// RenderCardLarge renders a single card as a 9×7 art block with rounded
// borders, corner rank+suit and a centered suit pip. Hidden cards render a
// patterned back.
func RenderCardLarge(c CardInfo) string {
	if isHidden(c) {
		return renderCardBox(frameRound, largeInnerCols, backLines(largeInnerCols, 5), styleCardBack, styleCardBack)
	}
	style := cardStyle(c.Suit)
	return renderCardBox(frameRound, largeInnerCols, largeCardLines(c), style, style)
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

// handArtWidth returns the total display width needed to render n cards at the
// given per-card width with gaps between them.
func handArtWidth(n, cardWidth int) int {
	if n <= 0 {
		return 0
	}
	return n*cardWidth + (n-1)*cardGap
}

// RenderHand renders multiple cards side by side, choosing the richest tier
// that fits in availableWidth: large art, then small art, then inline text.
func RenderHand(cards []CardInfo, availableWidth int) string {
	if len(cards) == 0 {
		return ""
	}

	if availableWidth < minWidthForArt {
		return RenderHandInline(cards)
	}

	n := len(cards)
	if handArtWidth(n, largeCardWidth) <= availableWidth {
		return RenderHandArtLarge(cards)
	}
	if handArtWidth(n, smallCardWidth) <= availableWidth {
		return RenderHandArt(cards)
	}
	return RenderHandInline(cards)
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

// RenderHandArt renders cards side by side as small (6×4) ASCII art blocks.
func RenderHandArt(cards []CardInfo) string {
	return renderHandTier(cards, RenderCard)
}

// RenderHandArtLarge renders cards side by side as large (9×7) art blocks.
func RenderHandArtLarge(cards []CardInfo) string {
	return renderHandTier(cards, RenderCardLarge)
}

// renderHandTier renders each card with the given renderer and joins them.
func renderHandTier(cards []CardInfo, render func(CardInfo) string) string {
	if len(cards) == 0 {
		return ""
	}
	rendered := make([]string, len(cards))
	for i, c := range cards {
		rendered[i] = render(c)
	}
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
