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
	assert.Equal(t, "1, 2, 3", game.GetAllowedTradeIns().String())

	testNoTrades := func(game *Game, err error) {
		assert.NoError(t, err)
		assert.Equal(t, "0", game.GetAllowedTradeIns().String())
		assert.True(t, game.CanTrade(0))
		assert.False(t, game.CanTrade(1))
		assert.False(t, game.CanTrade(2))
		assert.False(t, game.CanTrade(3))
		assert.False(t, game.CanTrade(4))
	}

	opts.TradeIns = []int{}
	testNoTrades(NewGame("", []int64{1, 2, 3}, opts))

	opts.TradeIns = []int{0}
	testNoTrades(NewGame("", []int64{1, 2, 3}, opts))
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
	assertHand(t, game, 2, "2c,5c,9c,12c")

	assert.NoError(t, trade(3, "4c,7c"))
	assertHand(t, game, 3, "3c,6c,10c,13c")

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

func TestGame_NextStage(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	assert.EqualError(t, game.NextStage(), "stage is not over")

	game.currentBet = 25
	game.decisionCount = 3
	game.idToParticipant[1].didFold = true
	assert.NoError(t, game.NextStage())

	assert.Equal(t, 0, game.currentBet)
	assert.Equal(t, 1, game.decisionCount)
	assert.Equal(t, 0, game.decisionStartIndex)
}

func TestGame_ParticipantAction(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	assert.NoError(t, game.DealCards())
	p := func(id int64) *Participant {
		return game.idToParticipant[id]
	}

	assert.Equal(t, 75, game.pot)
	game.community = deck.CardsFromString("2c,5h,4c")
	p(1).hand = deck.CardsFromString("14s,13s,12s") // this player will fold with the royal flush, silly-goose
	p(2).hand = deck.CardsFromString("3c,8d,10c")   // ends up with straight-flush
	p(3).hand = deck.CardsFromString("9c,9d,9h")    // loses with trips

	// trade-in round

	assert.NoError(t, game.tradeCardsForParticipant(p(1), []*deck.Card{}))
	assert.NoError(t, game.tradeCardsForParticipant(p(2), []*deck.Card{}))
	assert.NoError(t, game.tradeCardsForParticipant(p(3), []*deck.Card{}))
	assert.NoError(t, game.NextStage())

	// before first card is shown

	assert.Equal(t, []*deck.Card{nil, nil, nil}, game.GetCommunityCards())
	assert.Equal(t, ErrNotPlayersTurn, game.ParticipantChecks(p(2)))
	assert.Equal(t, ErrNotPlayersTurn, game.ParticipantFolds(p(2)))
	assert.Equal(t, ErrNotPlayersTurn, game.ParticipantBets(p(2), 25))
	assert.Equal(t, ErrNotPlayersTurn, game.ParticipantCalls(p(2)))

	assert.EqualError(t, game.ParticipantBets(p(1), game.pot+1), "your bet (76¢) must not exceed the current pot (75¢)")
	assert.NoError(t, game.ParticipantChecks(p(1)))
	assert.EqualError(t, game.ParticipantCalls(p(2)), "you cannot call without an active bet")
	assert.EqualError(t, game.ParticipantBets(p(2), 1), "your bet must at least match the ante (25¢)")
	assert.NoError(t, game.ParticipantBets(p(2), 75))
	assert.EqualError(t, game.ParticipantChecks(p(3)), "you cannot check with an active bet")
	assert.EqualError(t, game.ParticipantBets(p(3), 149), "your raise (149¢) must be at least equal to double the previous bet (150¢)")
	assert.NoError(t, game.ParticipantBets(p(3), 150))
	assert.NoError(t, game.ParticipantFolds(p(1)))
	assert.NoError(t, game.ParticipantCalls(p(2)))
	assert.Equal(t, ErrNotPlayersTurn, game.ParticipantCalls(p(3)))
	assert.NoError(t, game.NextStage())

	// before second card is shown

	assert.Equal(t, "2c,,", deck.CardsToString(game.GetCommunityCards()))
	assert.Equal(t, -25, p(1).balance)
	assert.Equal(t, -175, p(2).balance)
	assert.Equal(t, -175, p(3).balance)
	assert.Equal(t, 375, game.pot)

	assert.NoError(t, game.ParticipantBets(p(2), 25))
	assert.NoError(t, game.ParticipantBets(p(3), 50))
	assert.NoError(t, game.ParticipantBets(p(2), 100))
	assert.NoError(t, game.ParticipantCalls(p(3)))
	assert.NoError(t, game.NextStage())

	// before third card is shown

	assert.Equal(t, "2c,5h,", deck.CardsToString(game.GetCommunityCards()))
	assert.Equal(t, -25, p(1).balance)
	assert.Equal(t, -275, p(2).balance)
	assert.Equal(t, -275, p(3).balance)
	assert.Equal(t, 575, game.pot)

	assert.NoError(t, game.ParticipantChecks(p(2)))
	assert.NoError(t, game.ParticipantChecks(p(3)))
	assert.NoError(t, game.NextStage())

	// third card is now shown, final round of betting

	assert.Equal(t, "2c,5h,4c", deck.CardsToString(game.GetCommunityCards()))
	assert.Equal(t, -25, p(1).balance)
	assert.Equal(t, -275, p(2).balance)
	assert.Equal(t, -275, p(3).balance)
	assert.Equal(t, 575, game.pot)

	assert.NoError(t, game.ParticipantChecks(p(2)))
	assert.NoError(t, game.ParticipantChecks(p(3)))
	assert.False(t, game.IsGameOver())
	assert.NoError(t, game.NextStage())
	assert.True(t, game.IsGameOver())

	// XXX how to handle end of game
	// put in checks

	assert.Equal(t, ErrNotPlayersTurn, game.ParticipantChecks(p(2)))
	assert.Equal(t, -25, p(1).balance)
	assert.Equal(t, 300, p(2).balance) // won hand
	assert.Equal(t, -275, p(3).balance)
	assert.Equal(t, 575, game.pot)
	assert.Equal(t, 1, len(game.winners))
	assert.Equal(t, int64(2), game.winners[0].PlayerID)
}

func TestGame_ParticipantActionTie(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	assert.NoError(t, game.DealCards())
	p := func(id int64) *Participant {
		return game.idToParticipant[id]
	}

	assert.Equal(t, 75, game.pot)
	game.community = deck.CardsFromString("14s,5h,5s")
	p(1).hand = deck.CardsFromString("5c,6c,6d")
	p(2).hand = deck.CardsFromString("5d,6h,6s")
	p(3).hand = deck.CardsFromString("2c,4d,8h") // loses

	// trade-in round

	assert.NoError(t, game.tradeCardsForParticipant(p(1), []*deck.Card{}))
	assert.NoError(t, game.tradeCardsForParticipant(p(2), []*deck.Card{}))
	assert.NoError(t, game.tradeCardsForParticipant(p(3), []*deck.Card{}))
	assert.NoError(t, game.NextStage())

	for i := 0; i < 4; i++ {
		assert.NoError(t, game.ParticipantChecks(p(1)))
		assert.NoError(t, game.ParticipantChecks(p(2)))
		assert.NoError(t, game.ParticipantChecks(p(3)))
		assert.NoError(t, game.NextStage())
	}

	assert.Equal(t, 2, len(game.winners))
	assert.Equal(t, []int64{1, 2}, []int64{game.winners[0].PlayerID, game.winners[1].PlayerID})
	assert.Equal(t, 75, game.pot)
	assert.Equal(t, 13, p(1).balance)
	assert.Equal(t, 12, p(2).balance)
	assert.Equal(t, -25, p(3).balance)
}

func TestGame_ParticipantActionAllFold(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	assert.NoError(t, game.DealCards())
	p := func(id int64) *Participant {
		return game.idToParticipant[id]
	}

	assert.Equal(t, 75, game.pot)
	assert.NoError(t, game.ParticipantFolds(p(1)))
	assert.NoError(t, game.ParticipantFolds(p(2)))
	assert.Equal(t, ErrNotPlayersTurn, game.ParticipantFolds(p(3)))
	assert.True(t, game.IsGameOver())

	assert.Equal(t, 75, game.pot)
	assert.Equal(t, -25, p(1).balance)
	assert.Equal(t, -25, p(2).balance)
	assert.Equal(t, 50, p(3).balance)
}

func TestGame_FoldMidGame(t *testing.T) {
	opts := DefaultOptions()
	opts.Ante = 100
	game, _ := NewGame("", []int64{1, 2, 3, 4, 5}, opts)
	assert.NoError(t, game.DealCards())
	p := func(id int64) *Participant {
		return game.idToParticipant[id]
	}

	assert.NoError(t, game.tradeCardsForParticipant(p(1), []*deck.Card{}))
	assert.NoError(t, game.tradeCardsForParticipant(p(2), []*deck.Card{}))
	assert.NoError(t, game.tradeCardsForParticipant(p(3), []*deck.Card{}))
	assert.NoError(t, game.tradeCardsForParticipant(p(4), []*deck.Card{}))
	assert.NoError(t, game.tradeCardsForParticipant(p(5), []*deck.Card{}))
	assert.NoError(t, game.NextStage())

	assert.NoError(t, game.ParticipantBets(p(1), 200))
	assert.NoError(t, game.ParticipantFolds(p(2)))
	assert.NoError(t, game.ParticipantCalls(p(3)))
	assert.NoError(t, game.ParticipantCalls(p(4)))
	assert.NoError(t, game.ParticipantCalls(p(5)))
	assert.NoError(t, game.NextStage())

	assert.Equal(t, 1300, game.pot)

	assert.NoError(t, game.ParticipantChecks(p(1)))
	assert.NoError(t, game.ParticipantChecks(p(3)))
	assert.NoError(t, game.ParticipantChecks(p(4)))
	assert.NoError(t, game.ParticipantBets(p(5), 1200))
	assert.NoError(t, game.ParticipantFolds(p(1)))
	assert.NoError(t, game.ParticipantFolds(p(3)))
	assert.NoError(t, game.ParticipantFolds(p(4)))
	assert.True(t, game.IsGameOver())

	assert.Equal(t, 1, len(game.winners))

	assert.Equal(t, -300, p(1).balance)
	assert.Equal(t, -100, p(2).balance)
	assert.Equal(t, -300, p(3).balance)
	assert.Equal(t, -300, p(4).balance)
	assert.Equal(t, 1000, p(5).balance)
}

func TestGame_endOfStageAdjustments(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2}, DefaultOptions())
	p := func(id int64) *Participant {
		return game.idToParticipant[id]
	}
	assert.NoError(t, game.tradeCardsForParticipant(p(1), []*deck.Card{}))
	assert.NoError(t, game.tradeCardsForParticipant(p(2), []*deck.Card{}))
	game.endOfStageAdjustments()
	assert.PanicsWithValue(t, "already ran endOfStageAdjustments() for stage: 0", func() {
		_ = game.NextStage()
	})
}
