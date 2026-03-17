package texasholdem

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGame_Rules_Standard(t *testing.T) {
	game := setupNewGame(DefaultOptions(), 100, 100)

	rules := game.Rules()
	assert.True(t, len(rules) >= 4, "expected at least 4 rule sections")
	assert.Equal(t, "Overview", rules[0].Title)
	assert.Contains(t, rules[0].Body, "2 hole cards")
	assert.NotContains(t, rules[0].Body, "Pineapple")
}

func TestGame_Rules_Pineapple(t *testing.T) {
	opts := DefaultOptions()
	opts.Variant = Pineapple
	game := setupNewGame(opts, 100, 100)

	rules := game.Rules()
	assert.Contains(t, rules[0].Body, "3 hole cards")
	assert.Contains(t, rules[0].Body, "discard one card before the flop")
}

func TestGame_Rules_LazyPineapple(t *testing.T) {
	opts := DefaultOptions()
	opts.Variant = LazyPineapple
	game := setupNewGame(opts, 100, 100)

	rules := game.Rules()
	assert.Contains(t, rules[0].Body, "3 hole cards")
	assert.Contains(t, rules[0].Body, "Lazy Pineapple")
}

func TestGame_Rules_Blinds(t *testing.T) {
	game := setupNewGame(DefaultOptions(), 100, 100)

	rules := game.Rules()
	var blindsSection string
	for _, r := range rules {
		if r.Title == "Blinds & Ante" {
			blindsSection = r.Body
		}
	}
	assert.Contains(t, blindsSection, "Ante:")
	assert.Contains(t, blindsSection, "Small blind:")
	assert.Contains(t, blindsSection, "Big blind:")
}
