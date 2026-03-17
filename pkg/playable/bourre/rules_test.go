package bourre

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGame_Rules(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	assert.NoError(t, err)

	rules := g.Rules()
	assert.True(t, len(rules) >= 5, "expected at least 5 rule sections")
	assert.Equal(t, "Overview", rules[0].Title)
	assert.Contains(t, rules[0].Body, "5 cards")
}

func TestGame_Rules_FiveSuit(t *testing.T) {
	opts := DefaultOptions()
	opts.FiveSuit = true
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := g.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Five Suit" {
			found = true
			assert.Contains(t, r.Body, "five-suit deck")
		}
	}
	assert.True(t, found, "expected Five Suit section")
}

func TestGame_Rules_NoFiveSuit(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	assert.NoError(t, err)

	rules := g.Rules()
	for _, r := range rules {
		assert.NotEqual(t, "Five Suit", r.Title, "should not have Five Suit section in standard game")
	}
}
