package aceydeucey

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
	assert.Contains(t, rules[0].Body, "between")
}

func TestGame_Rules_ContinuousShoe(t *testing.T) {
	opts := DefaultOptions()
	opts.GameType = GameTypeContinuousShoe
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := g.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Continuous Shoe" {
			found = true
			assert.Contains(t, r.Body, "reshuffled")
		}
	}
	assert.True(t, found, "expected Continuous Shoe section")
}

func TestGame_Rules_AllowPass(t *testing.T) {
	opts := DefaultOptions()
	opts.AllowPass = true
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := g.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Passing" {
			found = true
		}
	}
	assert.True(t, found, "expected Passing section when AllowPass is true")
}

func TestGame_Rules_NoPass(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	assert.NoError(t, err)

	rules := g.Rules()
	for _, r := range rules {
		assert.NotEqual(t, "Passing", r.Title, "should not have Passing section in default game")
	}
}
