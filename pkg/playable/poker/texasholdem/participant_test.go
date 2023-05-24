package texasholdem

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
	"testing"
)

func TestGame_ActionsForParticipant(t *testing.T) {
	opts := Options{
		Variant:    Standard,
		Ante:       25,
		SmallBlind: 100,
		BigBlind:   200,
	}

	game := setupNewGame(opts, 1000, 1000, 1000, 1000)
	assertTick(t, game)

	a := assert.New(t)

	a.Nil(game.ActionsForParticipant(1))
	{
		assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound)

		a.Nil(game.ActionsForParticipant(1))
		a.Nil(game.ActionsForParticipant(2))
		a.Equal([]action.Action{action.Call, action.Raise, action.Fold}, game.ActionsForParticipant(3))
		a.Nil(game.ActionsForParticipant(4))

		assertAction(t, game, 3, action.Call)

		a.Equal([]action.Action{action.Call, action.Raise, action.Fold}, game.ActionsForParticipant(4))
		a.Nil(game.ActionsForParticipant(1))
		a.Nil(game.ActionsForParticipant(2))
		a.Nil(game.ActionsForParticipant(3))

		assertAction(t, game, 4, action.Call)
		assertAction(t, game, 1, action.Call)
		a.Equal([]action.Action{action.Check, action.Raise, action.Fold}, game.ActionsForParticipant(2))
		assertAction(t, game, 2, action.Check)

		a.True(game.potManager.IsRoundOver())
	}

	assertTickFromWaiting(t, game, DealerStateDealFlop)
	assertTick(t, game)

	a.Equal(DealerStateFlopBettingRound, game.dealerState)
	{
		a.Equal([]action.Action{action.Check, action.Bet, action.Fold}, game.ActionsForParticipant(1))
		assertAction(t, game, 1, action.Check)
		a.Equal([]action.Action{action.Check, action.Bet, action.Fold}, game.ActionsForParticipant(2))
		assertActionAndAmount(t, game, 2, action.Bet, 200)

		a.Equal([]action.Action{action.Call, action.Raise, action.Fold}, game.ActionsForParticipant(3))
		assertActionAndAmount(t, game, 3, action.Raise, game.participants[3].Balance()-25)

		a.Equal([]action.Action{action.Call, action.Raise, action.Fold}, game.ActionsForParticipant(4))
		assertActionAndAmount(t, game, 4, action.Raise, game.participants[4].Balance())

		a.Equal([]action.Action{action.Call, action.Fold}, game.ActionsForParticipant(1))
	}

	// pineapple discards
	{
		opts := Options{
			Variant:    Pineapple,
			Ante:       25,
			SmallBlind: 50,
			BigBlind:   100,
		}
		game := setupNewGame(opts, 25, 25, 25)
		assertTick(t, game)

		a.Equal([]action.Action{action.Discard}, game.ActionsForParticipant(1))
		a.Nil(game.ActionsForParticipant(2))
		a.Nil(game.ActionsForParticipant(3))
	}
}

func TestParticipant_getHandAnalyzer(t *testing.T) {
	a := assert.New(t)
	p := newParticipant(1, 100)
	p.getHandAnalyzer(nil)
	a.Nil(p.getHandAnalyzer(nil))

	p.cards = deck.CardsFromString("2c,2d,2h,2s,3c")
	a.NotNil(p.getHandAnalyzer(nil))
	a.Equal("Four of a kind", p.getHandAnalyzer(nil).GetHand().String())

	// override the hand analyzer to make sure caching still works
	p.handAnalyzer = handanalyzer.New(5, deck.CardsFromString("2c,2d,3c,3d,5c"))
	a.Equal("Two pair", p.getHandAnalyzer(nil).GetHand().String(), "cached value returned")

	a.Equal("Four of a kind", p.getHandAnalyzer(deck.CardsFromString("9d")).GetHand().String(), "cache is busted")
}

func TestParticipant_getHandAnalyzer_lazyPineapple(t *testing.T) {
	a := assert.New(t)

	p := newParticipant(1, 100)
	p.cards = deck.CardsFromString("2c,3c,4c")
	hand := p.getHandAnalyzer(deck.CardsFromString("4s,5c,6c,9s,10s")).GetHand().String()
	a.Equal("Straight", hand) // ensure not a flush

	// ensure we are using cards 1 and 3
	p.cards = deck.CardsFromString("2c,13d,4c")
	hand = p.getHandAnalyzer(deck.CardsFromString("3c,5c,6c,13s,13d")).GetHand().String()
	a.Equal("Straight flush", hand) // ensure not a flush

	// ensure we are using cards 2 and 3
	p.cards = deck.CardsFromString("13d,2c,4c")
	hand = p.getHandAnalyzer(deck.CardsFromString("3c,5c,6c,13s,13d")).GetHand().String()
	a.Equal("Straight flush", hand) // ensure not a flush
}

func TestParticipant_participantJSON(t *testing.T) {
	game := &Game{community: make(deck.Hand, 0)}
	p := &Participant{
		cards:  deck.CardsFromString("2c,3c"),
		reveal: false,
	}

	// cards are shown
	record := p.participantJSON(game, true)
	assert.Equal(t, "2c,3c", deck.CardsToString(record.Cards))

	// cards are null
	record = p.participantJSON(game, false)
	assert.Equal(t, ",", deck.CardsToString(record.Cards))

	p.reveal = true
	record = p.participantJSON(game, false)
	assert.NotNil(t, record.Cards)
}

func TestGame_FutureActionsForParticipant(t *testing.T) {
	a := assert.New(t)

	game := setupNewGame(DefaultOptions(), 1000, 1000, 1000, 1000)

	assertTick(t, game)
	assertTickFromWaiting(t, game, DealerStatePreFlopBettingRound)

	game.dealerState = DealerStateDealFlop
	a.Nil(game.FutureActionsForParticipant(1))

	game.dealerState = DealerStatePreFlopBettingRound
	a.Equal([]action.Action{action.Call, action.Fold}, game.FutureActionsForParticipant(1))
	a.Equal([]action.Action{action.Check, action.Fold}, game.FutureActionsForParticipant(2))
	a.Nil(game.FutureActionsForParticipant(3))
	a.Equal([]action.Action{action.Call, action.Fold}, game.FutureActionsForParticipant(4))

	game.dealerState = DealerStateRevealWinner
	a.Nil(game.FutureActionsForParticipant(1))

	// pineapple
	{
		opts := DefaultOptions()
		opts.Variant = Pineapple
		game := setupNewGame(opts, 1000, 1000, 1000)
		assertTick(t, game)
		a.Nil(game.FutureActionsForParticipant(1))
		a.Equal([]action.Action{action.Discard}, game.FutureActionsForParticipant(2))
		a.Equal([]action.Action{action.Discard}, game.FutureActionsForParticipant(3))
	}
}
