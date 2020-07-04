package littlel

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestGame_Name(t *testing.T) {
	g := &Game{}
	assert.Equal(t, "Little L", g.Name())
}

func TestGame_CanTrade(t *testing.T) {
	opts := DefaultOptions()
	game, err := NewGame("", []int64{1, 2, 3}, opts)
	assert.NoError(t, err)
	assert.True(t, game.CanTrade(0))
	assert.False(t, game.CanTrade(1))
	assert.True(t, game.CanTrade(2))
	assert.False(t, game.CanTrade(3))
	assert.False(t, game.CanTrade(4))

	opts.TradeIns = []int{3, 2, 3, 1}
	game, err = NewGame("", []int64{1, 2, 3}, opts)
	assert.NoError(t, err)
	assert.Equal(t, "1, 2, 3", game.GetAllowedTradeIns())
}

func TestNew(t *testing.T) {
	playerIDs := []int64{1, 2, 3}

	opts := Options{}
	game, err := NewGame("", playerIDs, opts)
	assert.EqualError(t, err, "ante must be greater than zero")
	assert.Nil(t, game)

	opts.Ante = 1
	game, err = NewGame("", playerIDs, opts)
	assert.EqualError(t, err, "the initial deal must be between 3 and 5 cards")
	assert.Nil(t, game)

	opts.InitialDeal = 4
	opts.TradeIns = []int{8}
	game, err = NewGame("", playerIDs, opts)
	assert.EqualError(t, err, "invalid trade-in option: 8")
	assert.Nil(t, game)

	opts.TradeIns = []int{0, 1, 2, 3, 4}

	game, err = NewGame("", []int64{1}, opts)
	assert.EqualError(t, err, "you must have at least two participants")
	assert.Nil(t, game)

	game, err = NewGame("", make([]int64, maxParticipants+1), opts)
	assert.EqualError(t, err, "you cannot have more than 10 participants")
	assert.Nil(t, game)

	game, err = NewGame("", playerIDs, opts)
	assert.NoError(t, err)
	assert.NotNil(t, game)
}

func TestGame_DealCards(t *testing.T) {
	game, err := NewGame("", []int64{1, 2}, DefaultOptions())
	game.deck = deck.New() // set deck to default, unshuffled deck

	assert.NoError(t, err)
	assert.NoError(t, game.DealCards())
	assert.Equal(t, "2c,4c,6c,8c", deck.CardsToString(game.idToParticipant[1].hand))
	assert.Equal(t, "3c,5c,7c,9c", deck.CardsToString(game.idToParticipant[2].hand))
	assert.Equal(t, "10c,11c,12c", deck.CardsToString(game.community))
}

func TestGame_TradeCardsForParticipant(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	game.deck = deck.New()
	trade := createTradeHelperFunc(game)
	assert.NoError(t, game.DealCards())

	game.stage = 1
	assert.EqualError(t, trade(2, "2c,5c"), "we are not in the trade-in stage")

	game.stage = 0
	assert.EqualError(t, trade(2, "2c,5c"), "it is not your turn")
	assert.NoError(t, trade(1, "2c,5c"))

	assert.EqualError(t, trade(2, "2c,5c"), "you do not have 2♣ in your hand")
	assert.EqualError(t, trade(2, "3c,3c"), "invalid trade-in")
	assert.EqualError(t, trade(2, "3c,6c,9c"), "the valid trade-ins are: 0, 2; you tried to trade 3")
	assert.NoError(t, trade(2, ""))

	assert.NoError(t, trade(3, "4c,7c"))
	assert.True(t, game.IsStageOver())
}

func TestGame_TradeCardsForParticipant_UsingDiscards(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	game.deck = deck.New()
	trade := createTradeHelperFunc(game)
	assert.NoError(t, game.DealCards())

	game.deck.Cards = deck.CardsFromString("10s,11s")
	rand.Seed(1)

	assert.NoError(t, trade(1, "2c,5c"))
	assertHand(t, game, 1, "8c,11c,10s,11s")

	assert.NoError(t, trade(2, "3c,6c"))
	assertHand(t, game, 2, "9c,12c,2c,5c")

	assert.NoError(t, trade(3, "4c,7c"))
	assertHand(t, game, 3, "10c,13c,3c,6c")

	assert.True(t, game.IsStageOver())
}

func assertHand(t *testing.T, game *Game, playerID int64, hand string) {
	t.Helper()

	assert.Equal(t, hand, deck.CardsToString(game.idToParticipant[playerID].hand))
}

func createTradeHelperFunc(game *Game) func(playerID int64, cards string) error {
	return func(playerID int64, cards string) error {
		p := game.idToParticipant[playerID]
		return game.tradeCardsForParticipant(p, deck.CardsFromString(cards))
	}
}

func TestGame_reset(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	game.currentBet = 25
	game.decisionIndex = 3
	game.idToParticipant[1].didFold = true
	game.reset()

	assert.Equal(t, 0, game.currentBet)
	assert.Equal(t, 1, game.decisionIndex)
}
