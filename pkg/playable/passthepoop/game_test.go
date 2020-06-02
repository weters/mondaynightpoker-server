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

	execOK, execError := createExecFunctions(t, game)

	execError(2, ActionTrade, "you are not up")
	execError(99, ActionTrade, "99 is not in this game")
	execError(1, GameAction(99), "not a valid game action")
	execOK(1, ActionTrade)
	// swap did not happen yet
	assert.Equal(t, card("2c"), participants[0].card)
	assert.Equal(t, card("3c"), participants[1].card)

	execError(2, ActionTrade, "there is a pending trade you have to accept")
	execError(2, ActionStay, "there is a pending trade you have to accept")
	execOK(2, ActionAccept)
	assert.Equal(t, card("3c"), participants[0].card)
	assert.Equal(t, card("2c"), participants[1].card)

	// ensure the first player cannot double trade
	execError(1, ActionTrade, "you are not up")
	execOK(2, ActionTrade)
	execError(3, ActionTrade, "there is a pending trade you have to accept")
	execOK(3, ActionAccept)
	assert.Equal(t, card("4c"), participants[1].card)
	assert.Equal(t, card("2c"), participants[2].card)

	// test going to the deck
	execError(3, ActionTrade, "the dealer can only go to the deck")
	execOK(3, ActionGoToDeck)
	execOK(3, ActionDrawFromDeck)
	assert.Equal(t, card("5c"), participants[2].card)

	execError(3, ActionStay, "no more decisions can be made this round")
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

	execOK, execError := createExecFunctions(t, game)

	// stay
	execOK(1, ActionStay)
	assert.Equal(t, card("10c"), participants[0].card)
	assert.Equal(t, card("2c"), participants[1].card)

	// hit a king
	execOK(2, ActionTrade)
	execError(3, ActionAccept, "you cannot accept the trade if you have a King")
	execError(3, ActionStay, "you have to flip the King")
	assert.Equal(t, card("2c"), participants[1].card)
	assert.Equal(t, card("13c"), participants[2].card)

	// cannot trade with king
	execError(3, ActionTrade, "you cannot trade a King")
	assert.Equal(t, card("13c"), participants[2].card)

	execError(3, ActionStay, "you have to flip the King")

	execOK(3, ActionFlipKing)
	assert.Equal(t, card("13c"), participants[2].card)
	assert.Equal(t, card("14c"), participants[3].card)

	// can trade for a king
	execOK(4, ActionGoToDeck)
	execOK(4, ActionDrawFromDeck)
	assert.Equal(t, card("13h"), participants[3].card)
}

func TestGame_ExecuteTurnForPlayer_DealerDeck(t *testing.T) {
	game, _ := NewGame("", []int64{1, 2}, DefaultOptions())
	execOK, execError := createExecFunctions(t, game)

	execError(1, ActionGoToDeck, "only the dealer may go to the deck")
	execOK(1, ActionStay)

	game.participants[1].card = card("13s")
	execError(2, ActionGoToDeck, "dealer must stay with a King")

	game.participants[1].card = card("2s") // ensure they don't have a King
	execError(2, ActionTrade, "the dealer can only go to the deck")
	execError(2, ActionDrawFromDeck, "you must first announce your intention to draw from the deck")

	execOK(2, ActionGoToDeck)

	execOK(2, ActionDrawFromDeck)
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

	execOK, _ := createExecFunctions(t, game)

	// round 1
	assert.EqualError(t, game.EndRound(), "not all players have had a turn yet")
	execOK(1, ActionStay)
	execOK(2, ActionStay)
	execOK(3, ActionStay)
	assert.NoError(t, game.EndRound())

	assert.True(t, game.shouldContinue())
	assert.NoError(t, game.nextRound())

	assert.Equal(t, 2, len(game.participants))
	assert.Equal(t, card("6d"), game.participants[0].card)
	assert.Equal(t, card("13d"), game.participants[1].card)

	execOK(2, ActionStay)
	execOK(3, ActionStay)

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
				Lives:           3,
				Pot:             game.options.Ante * 3,
				CardsLeftInDeck: 49,
				CurrentTurn:     3, // game rotated
				LastGameAction:  nil,
				LoserGroups:     nil,
			},
			Card:             card("9s"),
			AvailableActions: []GameAction{},
		},
	}, state)
}

func createExecFunctions(t *testing.T, game *Game) (func(playerID int64, action GameAction), func(playerID int64, action GameAction, expectedError string)) {
	t.Helper()

	execOK := func(playerID int64, action GameAction) {
		t.Helper()

		err := game.ExecuteTurnForPlayer(playerID, action)
		assert.NoError(t, err)
	}

	execError := func(playerID int64, action GameAction, expectedError string) {
		t.Helper()

		err := game.ExecuteTurnForPlayer(playerID, action)
		assert.EqualError(t, err, expectedError)
	}

	return execOK, execError
}
