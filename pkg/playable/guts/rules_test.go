package guts

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGame_Rules(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	assert.NoError(t, err)

	rules := g.Rules()
	assert.True(t, len(rules) >= 4, "expected at least 4 rule sections")
	assert.Equal(t, "Overview", rules[0].Title)
	assert.Contains(t, rules[0].Body, "2 cards")
}

func TestGame_Rules_ThreeCard(t *testing.T) {
	opts := DefaultOptions()
	opts.CardCount = 3
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := g.Rules()
	assert.Contains(t, rules[0].Body, "3 cards")
}

func TestGame_Rules_BloodyGuts(t *testing.T) {
	opts := DefaultOptions()
	opts.BloodyGuts = true
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := g.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Showdown" {
			assert.Contains(t, r.Body, "beat a hand drawn from the deck")
			found = true
		}
	}
	assert.True(t, found, "expected Showdown section")
}

func TestGame_Rules_AllowTrades(t *testing.T) {
	opts := DefaultOptions()
	opts.AllowTrades = true
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := g.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Trading" {
			found = true
		}
	}
	assert.True(t, found, "expected Trading section when AllowTrades is true")
}

func TestGame_Rules_NoTrades(t *testing.T) {
	opts := DefaultOptions()
	opts.AllowTrades = false
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := g.Rules()
	for _, r := range rules {
		assert.NotEqual(t, "Trading", r.Title, "should not have Trading section when AllowTrades is false")
	}
}
