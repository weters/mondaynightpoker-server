package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCardRankDisplay(t *testing.T) {
	tests := []struct {
		rank int
		want string
	}{
		{14, "A"},
		{13, "K"},
		{12, "Q"},
		{11, "J"},
		{10, "10"},
		{9, "9"},
		{2, "2"},
		{1, "?"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, cardRankDisplay(tt.rank))
	}
}

func TestSuitSymbol(t *testing.T) {
	assert.Equal(t, "♥", suitSymbol("hearts"))
	assert.Equal(t, "♦", suitSymbol("diamonds"))
	assert.Equal(t, "♣", suitSymbol("clubs"))
	assert.Equal(t, "♠", suitSymbol("spades"))
	assert.Equal(t, "stars", suitSymbol("stars"))
}

func TestIsRedSuit(t *testing.T) {
	assert.True(t, isRedSuit("hearts"))
	assert.True(t, isRedSuit("diamonds"))
	assert.False(t, isRedSuit("clubs"))
	assert.False(t, isRedSuit("spades"))
}

func TestRenderCard(t *testing.T) {
	card := CardInfo{Rank: 14, Suit: "hearts"}
	rendered := RenderCard(card)

	// Should have 4 lines
	lines := strings.Split(rendered, "\n")
	assert.Len(t, lines, 4)

	// Should contain the rank and suit symbol (with ANSI codes)
	assert.Contains(t, rendered, "A")
	assert.Contains(t, rendered, "♥")
	assert.Contains(t, rendered, "┌────┐")
	assert.Contains(t, rendered, "└────┘")
}

func TestRenderCard10(t *testing.T) {
	card := CardInfo{Rank: 10, Suit: "spades"}
	rendered := RenderCard(card)

	assert.Contains(t, rendered, "10")
	assert.Contains(t, rendered, "♠")
}

func TestRenderHandInline(t *testing.T) {
	cards := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "spades"},
	}
	result := RenderHandInline(cards)
	assert.Contains(t, result, "A♥")
	assert.Contains(t, result, "K♠")
}

func TestRenderHandEmpty(t *testing.T) {
	assert.Equal(t, "", RenderHand(nil, 100))
	assert.Equal(t, "", RenderHand([]CardInfo{}, 100))
}

func TestRenderHandNarrow(t *testing.T) {
	cards := []CardInfo{{Rank: 14, Suit: "hearts"}}
	result := RenderHand(cards, 30) // below minWidthForArt
	// Should use inline rendering
	assert.Contains(t, result, "A♥")
	assert.NotContains(t, result, "┌────┐")
}

func TestRenderHandWide(t *testing.T) {
	cards := []CardInfo{
		{Rank: 14, Suit: "hearts"},
		{Rank: 13, Suit: "spades"},
	}
	result := RenderHand(cards, 100) // above minWidthForArt
	// Should use art rendering
	assert.Contains(t, result, "┌────┐")
}

func TestRenderHandArtEmpty(t *testing.T) {
	assert.Equal(t, "", RenderHandArt(nil))
}

func TestIntersperse(t *testing.T) {
	assert.Equal(t, []string{"a"}, intersperse([]string{"a"}, " "))
	assert.Equal(t, []string{"a", " ", "b"}, intersperse([]string{"a", "b"}, " "))
	assert.Equal(t, []string{"a", "-", "b", "-", "c"}, intersperse([]string{"a", "b", "c"}, "-"))
}
