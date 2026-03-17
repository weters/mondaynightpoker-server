package sevencard

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGame_Rules_Stud(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	assert.NoError(t, err)

	rules := game.Rules()
	assert.True(t, len(rules) >= 3, "expected at least 3 rule sections")
	assert.Equal(t, "Overview", rules[0].Title)
	assert.Contains(t, rules[0].Body, "7 cards")
}

func TestGame_Rules_Baseball(t *testing.T) {
	opts := Options{Ante: 25, Variant: &Baseball{}}
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := game.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Baseball Rules" {
			found = true
			assert.Contains(t, r.Body, "3s and 9s are wild")
			assert.Contains(t, r.Body, "face-up 4")
		}
	}
	assert.True(t, found, "expected Baseball Rules section")
}

func TestGame_Rules_FollowTheQueen(t *testing.T) {
	opts := Options{Ante: 25, Variant: &FollowTheQueen{}}
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := game.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Follow the Queen Rules" {
			found = true
			assert.Contains(t, r.Body, "Queens are always wild")
		}
	}
	assert.True(t, found, "expected Follow the Queen Rules section")
}

func TestGame_Rules_LowCardWild(t *testing.T) {
	opts := Options{Ante: 25, Variant: &LowCardWild{}}
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := game.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Low Card Wild Rules" {
			found = true
			assert.Contains(t, r.Body, "lowest face-down card")
		}
	}
	assert.True(t, found, "expected Low Card Wild Rules section")
}

func TestGame_Rules_HighChicago(t *testing.T) {
	opts := Options{Ante: 25, Variant: &HighChicago{}}
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := game.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "High Chicago Rules" {
			found = true
			assert.Contains(t, r.Body, "highest spade in the hole")
		}
	}
	assert.True(t, found, "expected High Chicago Rules section")
}

func TestGame_Rules_CouponsAndClippings(t *testing.T) {
	opts := Options{Ante: 25, Variant: &CouponsAndClippings{}}
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := game.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Coupons and Clippings Rules" {
			found = true
			assert.Contains(t, r.Body, "BOGO")
		}
	}
	assert.True(t, found, "expected Coupons and Clippings Rules section")
}

func TestGame_Rules_Chiggs(t *testing.T) {
	opts := Options{Ante: 25, Variant: &Chiggs{}}
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	rules := game.Rules()
	found := false
	for _, r := range rules {
		if r.Title == "Chiggs Rules" {
			found = true
			assert.Contains(t, r.Body, "Mushroom")
		}
	}
	assert.True(t, found, "expected Chiggs Rules section")
}
