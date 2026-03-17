package passthepoop

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGame_Rules(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		Ante:    75,
		Lives:   3,
		Edition: &StandardEdition{},
	})
	assert.NoError(t, err)

	rules := g.Rules()
	assert.True(t, len(rules) >= 4, "expected at least 4 rule sections")
	assert.Equal(t, "Overview", rules[0].Title)
	assert.Contains(t, rules[0].Body, "3 lives")
	assert.Contains(t, rules[0].Body, "lowest card")
}

func TestGame_Rules_AllowBlocks(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		Ante:        75,
		Lives:       3,
		Edition:     &StandardEdition{},
		AllowBlocks: true,
	})
	assert.NoError(t, err)

	rules := g.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Blocks" {
			found = true
			assert.Contains(t, r.Body, "block")
		}
	}
	assert.True(t, found, "expected Blocks section when AllowBlocks is true")
}

func TestGame_Rules_PairsEdition(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		Ante:    75,
		Lives:   3,
		Edition: &PairsEdition{},
	})
	assert.NoError(t, err)

	rules := g.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Pairs Edition" {
			found = true
			assert.Contains(t, r.Body, "trips or better")
		}
	}
	assert.True(t, found, "expected Pairs Edition section")
}

func TestGame_Rules_DiarrheaEdition(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		Ante:    75,
		Lives:   3,
		Edition: &DiarrheaEdition{},
	})
	assert.NoError(t, err)

	rules := g.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Diarrhea Edition" {
			found = true
			assert.Contains(t, r.Body, "extra life")
		}
	}
	assert.True(t, found, "expected Diarrhea Edition section")
}

func TestGame_Rules_NoBlocks(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		Ante:        75,
		Lives:       3,
		Edition:     &StandardEdition{},
		AllowBlocks: false,
	})
	assert.NoError(t, err)

	rules := g.Rules()
	for _, r := range rules {
		assert.NotEqual(t, "Blocks", r.Title, "should not have Blocks section when AllowBlocks is false")
	}
}
