package passthepoop

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func TestGame_Name(t *testing.T) {
	game, err := NewGame(logrus.StandardLogger(), nil, Options{})
	assert.EqualError(t, err, "game requires at least two players")
	assert.Nil(t, game)

	players := []int64{0, 1}

	game, err = NewGame(logrus.StandardLogger(), players, Options{})
	assert.EqualError(t, err, "ante must be greater than 0")
	assert.Nil(t, game)

	game, err = NewGame(logrus.StandardLogger(), players, Options{Ante: 25})
	assert.EqualError(t, err, "lives must be greater than 0")
	assert.Nil(t, game)

	game, err = NewGame(logrus.StandardLogger(), players, DefaultOptions())
	assert.NoError(t, err)
	assert.Equal(t, "Pass the Poop, Standard Edition", game.Name())

	opts := DefaultOptions()
	opts.Edition = &PairsEdition{}
	game, _ = NewGame(logrus.StandardLogger(), players, opts)
	assert.Equal(t, "Pass the Poop, Pairs Edition", game.Name())
}

func Test_nextRound(t *testing.T) {
	ids := []int64{1, 2, 3, 4, 5}
	game, err := NewGame(logrus.StandardLogger(), ids, DefaultOptions())
	assert.NoError(t, err)
	participants := game.participants

	execOK, _ := createExecFunctions(t, game)
	stay := func() {
		for _, p := range game.participants {
			execOK(p.PlayerID, ActionStay)
		}
	}

	dealCards(game, "2c", "3c", "4c", "5c", "6c")
	stay()
	assert.NoError(t, game.EndRound())
	assert.NoError(t, game.nextRound())
	assert.Equal(t, []int64{2, 3, 4, 5, 1}, getPlayerIDsFromGame(game))

	dealCards(game, "2c", "3c", "4c", "5c", "6c")
	stay()
	assert.NoError(t, game.EndRound())
	assert.NoError(t, game.nextRound())
	assert.Equal(t, []int64{3, 4, 5, 1, 2}, getPlayerIDsFromGame(game))

	dealCards(game, "2c", "2c", "4c", "5c", "6c")
	game.participants[0].lives = 1
	game.participants[1].lives = 1
	stay()
	assert.NoError(t, game.EndRound())
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
	dealCards(game, "2c", "2c", "3c")
	stay()
	assert.NoError(t, game.EndRound())
	assert.EqualError(t, game.nextRound(), "not enough players for a new round")
}

func TestGame_ExecuteTurnForPlayer_AllTrades(t *testing.T) {
	ids := []int64{1, 2, 3}
	game, _ := NewGame(logrus.StandardLogger(), ids, DefaultOptions())
	participants := game.participants
	participants[0].card = card("2c")
	participants[1].card = card("3c")
	participants[2].card = card("4c")
	game.deck.Cards[0] = card("5c")

	execOK, execError := createExecFunctions(t, game)

	execError(2, ActionTrade, "you are not up")
	execError(99, ActionTrade, "99 is not in this game")
	// an out-of-range action can no longer come through Game.Action(), so exercise it directly
	assert.EqualError(t, game.ExecuteTurnForPlayer(1, GameAction(99)), "not a valid game action")
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

func TestGame_ExecuteTurnForPlayer_WithBlocks(t *testing.T) {
	// first, make sure that blocks are only allowed in games with blocks
	opts := DefaultOptions()
	opts.AllowBlocks = false
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)

	game.idToParticipant[1].card = deck.CardFromString("2c")
	game.idToParticipant[2].card = deck.CardFromString("3c")

	execOK, execErr := createExecFunctions(t, game)
	execOK(1, ActionTrade)
	execErr(2, ActionBlockTrade, "blocks are not allowed")

	// now test the actual blocks

	opts.AllowBlocks = true
	game, _ = NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, opts)

	game.idToParticipant[1].card = deck.CardFromString("2c")
	game.idToParticipant[2].card = deck.CardFromString("3c")
	game.idToParticipant[3].card = deck.CardFromString("4c")
	game.idToParticipant[4].card = deck.CardFromString("5c")

	execOK, execErr = createExecFunctions(t, game)
	execOK(1, ActionTrade)
	execOK(2, ActionBlockTrade)

	// blocking rejects the incoming trade but does NOT consume the blocker's
	// own turn: player 2 keeps their card and is still up to make a decision
	execErr(3, ActionTrade, "you are not up")
	execErr(2, ActionBlockTrade, "there is not a pending trade to block")
	execOK(2, ActionTrade) // player 2 now trades into player 3

	// player 3 can, in turn, block player 2's trade and keep their own turn
	execOK(3, ActionBlockTrade)
	execErr(4, ActionTrade, "you are not up")
	execOK(3, ActionTrade) // player 3 trades into player 4 (the dealer)

	// player 4 (dealer) has no block chip, so they cannot block the trade
	game.idToParticipant[4].hasBlock = false
	execErr(4, ActionBlockTrade, "you do not have a block")
}

func TestGame_BlockTrade_KeepsTurnAndLogsBlock(t *testing.T) {
	opts := DefaultOptions()
	opts.AllowBlocks = true
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)

	dealCards(game, "5c", "6c", "7c")
	game.deck.Cards[0] = card("8c")

	// discard the "new game" log message so we only inspect the block below
	drainLog(game)

	execOK, _ := createExecFunctions(t, game)

	execOK(1, ActionTrade)      // player 1 trades into player 2
	execOK(2, ActionBlockTrade) // player 2 blocks

	msgs := drainLog(game)

	// the trade and the block should each be logged, and the block must be
	// attributed to the blocker (player 2) — not logged as a "stay"
	var tradeMsg, blockMsg *playable.LogMessage
	for _, m := range msgs {
		switch m.Message {
		case "{} trades their card":
			tradeMsg = m
		case "{} blocked the trade":
			blockMsg = m
		}
		assert.NotContains(t, m.Message, "will stay", "a block must never be logged as a stay")
	}

	if assert.NotNil(t, tradeMsg, "the trade should be logged") {
		assert.Equal(t, []int64{1}, tradeMsg.PlayerIDs)
	}
	if assert.NotNil(t, blockMsg, "the block should be logged and attributed to the blocker") {
		assert.Equal(t, []int64{2}, blockMsg.PlayerIDs)
	}

	// the last game action records both the blocker and the blocked trader so
	// the client can render "player 1 was blocked by player 2"
	assert.Equal(t, ActionBlockTrade, game.lastGameAction.GameAction)
	assert.Equal(t, int64(2), game.lastGameAction.PlayerID)
	assert.Equal(t, int64(1), game.lastGameAction.SecondaryPlayerID)

	// blocking does not consume the blocker's turn: player 2 is still up
	assert.Equal(t, game.idToParticipant[2], game.getCurrentTurn())
	assert.False(t, game.pendingTrade)

	// player 2 can now make their own decision, which advances play to player 3
	execOK(2, ActionStay)
	assert.Equal(t, game.idToParticipant[3], game.getCurrentTurn())
}

func TestGame_BlockTrade_DealerKeepsTurn(t *testing.T) {
	opts := DefaultOptions()
	opts.AllowBlocks = true
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)

	dealCards(game, "5c", "6c")
	game.deck.Cards[0] = card("8c")

	execOK, _ := createExecFunctions(t, game)

	// player 1 trades into player 2, who is the dealer (last player)
	execOK(1, ActionTrade)
	execOK(2, ActionBlockTrade)

	// blocking must NOT end the round: the dealer still owns their turn and can
	// go to the deck. Before the fix, the block advanced past the dealer and the
	// round ended prematurely, which left the game looking stuck to clients.
	assert.Equal(t, game.idToParticipant[2], game.getCurrentTurn())
	assert.Nil(t, game.loserGroups, "the round must not have ended")

	actions := game.getActionsForParticipant(game.idToParticipant[2])
	assert.Equal(t, []GameAction{ActionStay, ActionGoToDeck}, actions)

	// the dealer can complete their turn, after which the round ends normally
	execOK(2, ActionStay)
	assert.Nil(t, game.getCurrentTurn())
	assert.NoError(t, game.EndRound())
}

func TestGame_ExecuteTurnForPlayer_KingedAndStays(t *testing.T) {
	ids := []int64{1, 2, 3, 4}
	game, _ := NewGame(logrus.StandardLogger(), ids, DefaultOptions())
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
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
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
	game, _ := NewGame(logrus.StandardLogger(), ids, DefaultOptions())
	game.flipAllCards()

	for i := 0; i < 4; i++ {
		assert.True(t, game.participants[i].isFlipped)
	}
}

func TestGame_CompleteGame(t *testing.T) {
	opts := DefaultOptions()
	opts.Lives = 1
	opts.Seed = 1

	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
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
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())
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

func TestGame_NextRoundAndEndRound(t *testing.T) {
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, DefaultOptions())
	assert.EqualError(t, game.nextRound(), "you must end the round first")

	execOk, _ := createExecFunctions(t, game)
	dealCards(game, "3c", "4c", "3c")
	execOk(1, ActionStay)
	execOk(2, ActionStay)
	execOk(3, ActionStay)
	assert.NoError(t, game.EndRound())
	assert.EqualError(t, game.EndRound(), "you cannot end the round multiple times")
	livesEqual(t, game, map[int64]int{
		1: 2,
		2: 3,
		3: 2,
	})

	assert.NoError(t, game.nextRound())
	assert.EqualError(t, game.nextRound(), "you must end the round first")

	assert.Equal(t, []int64{2, 3, 1}, getPlayerIDsFromGame(game))
}

func createExecFunctions(t *testing.T, game *Game) (func(playerID int64, action GameAction), func(playerID int64, action GameAction, expectedError string)) {
	t.Helper()

	execOK := func(playerID int64, action GameAction) {
		t.Helper()

		resp, updateState, err := game.Action(playerID, &playable.PayloadIn{Action: action.ID()})
		assert.NoError(t, err)
		assert.Equal(t, playable.OK(), resp)
		assert.True(t, updateState)
	}

	execError := func(playerID int64, action GameAction, expectedError string) {
		t.Helper()

		resp, updateState, err := game.Action(playerID, &playable.PayloadIn{Action: action.ID()})
		assert.EqualError(t, err, expectedError)
		assert.Nil(t, resp)
		assert.False(t, updateState)
	}

	return execOK, execError
}

func TestGame_getActionsForParticipant(t *testing.T) {
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		Ante:    100,
		Lives:   2,
		Edition: &StandardEdition{},
	})

	execOK, _ := createExecFunctions(t, game)

	game.idToParticipant[1].card = card("2c")
	actions := game.getActionsForParticipant(game.idToParticipant[1])
	assert.Equal(t, []GameAction{
		ActionStay,
		ActionTrade,
	}, actions)

	game.idToParticipant[1].card = card("13c")
	actions = game.getActionsForParticipant(game.idToParticipant[1])
	assert.Equal(t, []GameAction{
		ActionStay,
		ActionFlipKing,
	}, actions)

	// no actions for player out of turn
	assert.Equal(t, []GameAction{}, game.getActionsForParticipant(game.idToParticipant[2]))

	game.idToParticipant[1].card = card("2c")
	execOK(1, ActionTrade)

	game.idToParticipant[2].card = card("13c")
	actions = game.getActionsForParticipant(game.idToParticipant[2])
	assert.Equal(t, []GameAction{
		ActionFlipKing,
	}, actions)

	game.idToParticipant[2].card = card("12c")
	actions = game.getActionsForParticipant(game.idToParticipant[2])
	assert.Equal(t, []GameAction{
		ActionAccept,
	}, actions)

	execOK(2, ActionAccept)

	game.idToParticipant[2].card = card("13c")
	actions = game.getActionsForParticipant(game.idToParticipant[2])
	assert.Equal(t, []GameAction{
		ActionStay,
		ActionFlipKing,
	}, actions)

	game.idToParticipant[2].card = card("12c")
	actions = game.getActionsForParticipant(game.idToParticipant[2])
	assert.Equal(t, []GameAction{
		ActionStay,
		ActionGoToDeck,
	}, actions)

	execOK(2, ActionGoToDeck)

	actions = game.getActionsForParticipant(game.idToParticipant[2])
	assert.Equal(t, []GameAction{
		ActionDrawFromDeck,
	}, actions)

	execOK(2, ActionDrawFromDeck)

	actions = game.getActionsForParticipant(game.idToParticipant[1])
	assert.Equal(t, []GameAction{}, actions)
	actions = game.getActionsForParticipant(game.idToParticipant[2])
	assert.Equal(t, []GameAction{}, actions)

	game.idToParticipant[1].card = card("3c")
	game.idToParticipant[2].card = card("4c")
	assert.NoError(t, game.EndRound())

	actions = game.getActionsForParticipant(game.idToParticipant[2])
	assert.Equal(t, []GameAction{}, actions)

	assert.NoError(t, game.nextRound())

	// double check we are in a known state
	assert.Equal(t, 1, game.idToParticipant[1].lives)
	assert.Equal(t, 2, game.idToParticipant[2].lives)

	game.idToParticipant[1].card = card("2c")
	game.idToParticipant[2].card = card("3c")

	execOK(2, ActionStay)
	execOK(1, ActionStay)
	assert.NoError(t, game.EndRound())

	actions = game.getActionsForParticipant(game.idToParticipant[2])
	assert.Equal(t, []GameAction{}, actions)

	assert.True(t, game.isGameOver())
}

func TestGame_getActionsForParticipantWithBlocks(t *testing.T) {
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		Ante:        100,
		Lives:       2,
		Edition:     &StandardEdition{},
		AllowBlocks: true,
	})

	a := assert.New(t)
	a.True(game.idToParticipant[1].hasBlock)
	a.True(game.idToParticipant[2].hasBlock)

	execOK, _ := createExecFunctions(t, game)
	execOK(1, ActionTrade)

	game.idToParticipant[2].card = card("12c")
	actions := game.getActionsForParticipant(game.idToParticipant[2])
	a.Equal([]GameAction{ActionBlockTrade, ActionAccept}, actions)

	game.idToParticipant[2].hasBlock = false
	actions = game.getActionsForParticipant(game.idToParticipant[2])
	a.Equal([]GameAction{ActionAccept}, actions)
}

func TestGame_Action_unknownAction(t *testing.T) {
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())

	resp, updateState, err := game.Action(1, &playable.PayloadIn{Action: "not-a-real-action"})
	assert.EqualError(t, err, "unsupported action: not-a-real-action")
	assert.Nil(t, resp)
	assert.False(t, updateState)
}
