package passthepoop

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func TestGame_Name(t *testing.T) {
	game, err := NewGame("", nil, Options{})
	assert.EqualError(t, err, "game requires at least two players")
	assert.Nil(t, game)

	players := []int64{0, 1}

	game, err = NewGame("", players, Options{})
	assert.EqualError(t, err, "ante must be greater than 0")
	assert.Nil(t, game)

	game, err = NewGame("", players, Options{Ante: 25})
	assert.EqualError(t, err, "lives must be greater than 0")
	assert.Nil(t, game)

	game, err = NewGame("", players, DefaultOptions())
	assert.NoError(t, err)
	assert.Equal(t, "Pass the Poop, Standard Edition", game.Name())

	opts := DefaultOptions()
	opts.Edition = &PairsEdition{}
	game, _ = NewGame("", players, opts)
	assert.Equal(t, "Pass the Poop, Pairs Edition", game.Name())
}

func Test_nextRound(t *testing.T) {
	ids := []int64{1, 2, 3, 4, 5}
	game, err := NewGame("", ids, DefaultOptions())
	assert.NoError(t, err)
	participants := game.participants

	assert.NoError(t, game.nextRound())
	assert.Equal(t, []int64{2, 3, 4, 5, 1}, getPlayerIDsFromGame(game))

	assert.NoError(t, game.nextRound())
	assert.Equal(t, []int64{3, 4, 5, 1, 2}, getPlayerIDsFromGame(game))

	game.participants[0].lives = 0 // id=3, lost
	game.participants[1].lives = 0 // id=4, lost
	game.participants[2].lives = 1 // id=5, still alive

	assert.NoError(t, game.nextRound())
	assert.Equal(t, []int64{5, 1, 2}, getPlayerIDsFromGame(game))
	assert.Equal(t, map[int64]*Participant{
		1: participants[0],
		2: participants[1],
		3: participants[2],
		4: participants[3],
		5: participants[4],
	}, game.idToParticipant)

	game.participants[0].lives = 0 // id=5
	game.participants[1].lives = 0 // id=1

	assert.EqualError(t, game.nextRound(), "not enough players for a new round")
}

func TestGame_ExecuteTurnForPlayer_AllTrades(t *testing.T) {
	ids := []int64{1, 2, 3}
	game, _ := NewGame("", ids, DefaultOptions())
	participants := game.participants
	participants[0].card = card("2c")
	participants[1].card = card("3c")
	participants[2].card = card("4c")
	game.deck.Cards[0] = card("5c")

	res, err := game.ExecuteTurnForPlayer(2, ActionTrade)
	assert.EqualError(t, err, "you are not up")
	assert.Equal(t, ResultError, res)

	res, err = game.ExecuteTurnForPlayer(99, ActionTrade)
	assert.EqualError(t, err, "99 is not in this game")
	assert.Equal(t, ResultError, res)

	res, err = game.ExecuteTurnForPlayer(1, GameAction(99))
	assert.EqualError(t, err, "not a valid game action")
	assert.Equal(t, ResultError, res)

	res, err = game.ExecuteTurnForPlayer(1, ActionTrade)
	assert.NoError(t, err)
	assert.Equal(t, ResultOK, res)
	assert.Equal(t, card("3c"), participants[0].card)
	assert.Equal(t, card("2c"), participants[1].card)

	// ensure the first player cannot double trade
	res, err = game.ExecuteTurnForPlayer(1, ActionTrade)
	assert.EqualError(t, err, "you are not up")
	assert.Equal(t, ResultError, res)

	res, err = game.ExecuteTurnForPlayer(2, ActionTrade)
	assert.NoError(t, err)
	assert.Equal(t, ResultOK, res)
	assert.Equal(t, card("4c"), participants[1].card)
	assert.Equal(t, card("2c"), participants[2].card)

	// test going to the deck
	res, err = game.ExecuteTurnForPlayer(3, ActionGoToDeck)
	assert.NoError(t, err)
	assert.Equal(t, ResultOK, res)
	res, err = game.ExecuteTurnForPlayer(3, ActionTrade)
	assert.NoError(t, err)
	assert.Equal(t, ResultOK, res)
	assert.Equal(t, card("5c"), participants[2].card)

	res, err = game.ExecuteTurnForPlayer(3, ActionTrade)
	assert.EqualError(t, err, "no more decisions can be made this round")
	assert.Equal(t, ResultError, res)
}

func TestGame_ExecuteTurnForPlayer_KingedAndStays(t *testing.T) {
	ids := []int64{1, 2, 3, 4}
	game, _ := NewGame("", ids, DefaultOptions())
	participants := game.participants
	participants[0].card = card("10c")
	participants[1].card = card("2c")
	participants[2].card = card("13c")
	participants[3].card = card("14c")
	game.deck.Cards[0] = card("13h")

	// stay
	res, err := game.ExecuteTurnForPlayer(1, ActionStay)
	assert.NoError(t, err)
	assert.Equal(t, ResultOK, res)
	assert.Equal(t, card("10c"), participants[0].card)
	assert.Equal(t, card("2c"), participants[1].card)

	// hit a king
	res, err = game.ExecuteTurnForPlayer(2, ActionTrade)
	assert.NoError(t, err)
	assert.Equal(t, ResultKing, res)
	assert.Equal(t, card("2c"), participants[1].card)
	assert.Equal(t, card("13c"), participants[2].card)

	// cannot trade with king
	res, err = game.ExecuteTurnForPlayer(3, ActionTrade)
	assert.EqualError(t, err, "you cannot trade a king")
	assert.Equal(t, ResultError, res)
	assert.Equal(t, card("13c"), participants[2].card)

	res, err = game.ExecuteTurnForPlayer(3, ActionStay)
	assert.NoError(t, err)
	assert.Equal(t, ResultOK, res)
	assert.Equal(t, card("13c"), participants[2].card)
	assert.Equal(t, card("14c"), participants[3].card)

	// can trade for a king
	res, err = game.ExecuteTurnForPlayer(4, ActionGoToDeck)
	assert.NoError(t, err)
	assert.Equal(t, ResultOK, res)
	res, err = game.ExecuteTurnForPlayer(4, ActionTrade)
	assert.NoError(t, err)
	assert.Equal(t, ResultOK, res)
	assert.Equal(t, card("13h"), participants[3].card)
}

func TestGame_ExecuteTurnForPlayer_DealerDeck(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2}, DefaultOptions())
	res, err := game.ExecuteTurnForPlayer(1, ActionGoToDeck)
	assert.Equal(t, ResultError, res)
	assert.EqualError(t, err, "only the dealer may go to the deck")

	res, err = game.ExecuteTurnForPlayer(1, ActionStay)
	assert.Equal(t, ResultOK, res)
	assert.NoError(t, err)

	res, err = game.ExecuteTurnForPlayer(2, ActionTrade)
	assert.Equal(t, ResultError, res)
	assert.EqualError(t, err, "you must first announce your intention to go to the deck")

	res, err = game.ExecuteTurnForPlayer(2, ActionGoToDeck)
	assert.Equal(t, ResultOK, res)
	assert.NoError(t, err)

	res, err = game.ExecuteTurnForPlayer(2, ActionTrade)
	assert.Equal(t, ResultOK, res)
	assert.NoError(t, err)
}

func getPlayerIDsFromGame(g *Game) []int64 {
	ids := make([]int64, len(g.participants))
	for i, p := range g.participants {
		ids[i] = p.PlayerID
	}

	return ids
}

func TestGame_flipAllCards(t *testing.T) {
	ids := []int64{1, 2, 3, 4}
	game, _ := NewGame("", ids, DefaultOptions())
	game.flipAllCards()

	for i := 0; i < 4; i++ {
		assert.True(t, game.participants[i].isFlipped)
	}
}

func TestGame_CompleteGame(t *testing.T) {
	opts := DefaultOptions()
	opts.Lives = 1
	seed = 1
	game, err := NewGame("", []int64{1, 2, 3}, opts)
	assert.NoError(t, err)
	game.participants[0].card = card("2c")
	game.participants[1].card = card("3c")
	game.participants[2].card = card("4c")

	execOK := func(res ActionResult, err error) {
		t.Helper()
		assert.Equal(t, ResultOK, res)
		assert.NoError(t, err)
	}

	// round 1
	execOK(game.ExecuteTurnForPlayer(1, ActionStay))
	execOK(game.ExecuteTurnForPlayer(2, ActionStay))
	execOK(game.ExecuteTurnForPlayer(3, ActionStay))
	assert.NoError(t, game.EndRound())

	assert.True(t, game.shouldContinue())
	assert.NoError(t, game.nextRound())

	assert.Equal(t, 2, len(game.participants))
	assert.Equal(t, card("6d"), game.participants[0].card)
	assert.Equal(t, card("13d"), game.participants[1].card)

	execOK(game.ExecuteTurnForPlayer(2, ActionStay))
	execOK(game.ExecuteTurnForPlayer(3, ActionStay))

	assert.NoError(t, game.EndRound())
	assert.False(t, game.shouldContinue())
}

func TestGame_GetPlayerState(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	p1 := game.participants[0]
	p1.card = card("9s")
	game.participants[1].lives = 0
	game.eliminateAndRotateParticipants()

	state, err := game.GetPlayerState(1)
	assert.NoError(t, err)
	assert.Equal(t, &playable.Response{
		Key:   "game",
		Value: "pass-the-poop",
		Data: &ParticipantState{
			Participant: p1,
			GameState: &GameState{
				Edition:         "Standard",
				Participants:    game.participants,
				AllParticipants: game.idToParticipant,
				Ante:            game.options.Ante,
				Pot:             game.options.Ante * 3,
				DecisionIndex:   0,
			},
			Card: card("9s"),
		},
	}, state)
}
