package littlel

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGame_Rules(t *testing.T) {
	game, err := NewGameV2(logrus.StandardLogger(), newParticipants(100, 100), DefaultOptions())
	assert.NoError(t, err)

	rules := game.Rules()
	assert.True(t, len(rules) >= 4, "expected at least 4 rule sections")
	assert.Equal(t, "Overview", rules[0].Title)
	assert.Contains(t, rules[0].Body, "4 cards")

	tradeFound := false
	for _, r := range rules {
		if r.Title == "Trade-In" {
			tradeFound = true
			assert.Contains(t, r.Body, "trade")
		}
	}
	assert.True(t, tradeFound, "expected Trade-In section")
}

func TestGame_Rules_FiveCard(t *testing.T) {
	opts := DefaultOptions()
	opts.InitialDeal = 5
	game, err := NewGameV2(logrus.StandardLogger(), newParticipants(100, 100), opts)
	assert.NoError(t, err)

	rules := game.Rules()
	assert.Contains(t, rules[0].Body, "5 cards")
}
