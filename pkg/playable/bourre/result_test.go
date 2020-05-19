package bourre

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestResult_ShouldContinue(t *testing.T) {
	assert.False(t, (&Result{NewPot: 0}).ShouldContinue())
	assert.True(t, (&Result{NewPot: 1}).ShouldContinue())
}

func TestResult_NewGame(t *testing.T) {
	r := &Result{NewPot: 0}
	g, err := r.NewGame()
	assert.Equal(t, ErrCannotCreateGame, err)
	assert.Nil(t, g)

	p1 := &Player{balance: 0, folded: false, winCount: 0}
	p2 := &Player{balance: 0, folded: false, winCount: 1}
	p3 := &Player{balance: 0, folded: false, winCount: 2, hand: cardsFromString("2c,2s")}
	p4 := &Player{balance: 0, folded: false, winCount: 1}
	p5 := &Player{balance: 0, folded: false, winCount: 1}
	p6 := &Player{balance: 0, folded: true, winCount: 0}
	p7 := &Player{balance: 0, folded: true, winCount: 0}

	playerOrder := map[*Player]int{
		p1: 0,
		p2: 1,
		p3: 2,
		p4: 3,
		p5: 4,
		p6: 5,
		p7: 6,
	}

	idToPlayer := map[int64]*Player{
		10: &Player{},
		20: &Player{},
	}
	r = &Result{
		Booted:      []*Player{p7},
		Ante:        5,
		NewPot:      50,
		playerOrder: playerOrder,
		idToPlayer:  idToPlayer,
	}

	g, err = r.NewGame()
	assert.NoError(t, err)
	assert.NotNil(t, g)

	assert.Equal(t, idToPlayer, g.idToPlayer)
	assert.Equal(t, 5, g.ante)

	assert.Equal(t, 5, len(g.playerOrder), "removed two players")
	assert.Equal(t, 7, len(playerOrder), "original not touched")

	assert.Equal(t, 4, g.playerOrder[p1]) // moved to the end
	assert.Equal(t, 0, g.playerOrder[p2])
	assert.Equal(t, 1, g.playerOrder[p3])
	assert.Equal(t, 2, g.playerOrder[p4])
	assert.Equal(t, 3, g.playerOrder[p5])

	// ensure player is reset
	assert.Equal(t, 0, len(p3.hand))
	assert.False(t, p3.folded)
	assert.Equal(t, 0, p3.winCount)
}

func TestResult_NewGame_Booted(t *testing.T) {
	game, players := setupGame("14S", []string{
		"14c,14d,13h,12s,2s",
		"13c,13d,14h,14s,3s",
		"12c,12d,12h,13s,4s",
	})

	players[0].PlayerID = 0
	players[0].balance = -50
	players[1].PlayerID = 1
	players[1].balance = -50
	players[2].PlayerID = 2
	players[2].balance = -50

	game.pot = 150
	game.ante = 50

	cardFunc := createPlayCardFunc(t, game, players)
	game.playerDidDiscard(players[0], []*deck.Card{})
	game.playerDidDiscard(players[1], []*deck.Card{})
	game.playerDidDiscard(players[2], []*deck.Card{})
	assert.NoError(t, game.replaceDiscards())

	cardFunc(0, 0)
	cardFunc(1, 0)
	cardFunc(2, 0)
	assert.NoError(t, game.nextRound())

	cardFunc(1, 0)
	cardFunc(2, 0)
	cardFunc(0, 0)
	assert.NoError(t, game.nextRound())

	cardFunc(2, 0)
	cardFunc(0, 0)
	cardFunc(1, 0)
	assert.NoError(t, game.nextRound())

	cardFunc(0, 0)
	cardFunc(1, 0)
	cardFunc(2, 0)
	assert.NoError(t, game.nextRound())

	cardFunc(1, 0)
	cardFunc(2, 0)
	cardFunc(0, 0)
	assert.NoError(t, game.nextRound())

	res := game.result
	assert.NotNil(t, res)

	assert.Equal(t, 1, len(res.Booted))
	assert.Equal(t, players[2], res.Booted[0])

	game, err := res.NewGame()
	assert.NoError(t, err)
	players[0].hand = cardsFromString("2c,3c,4c,5c,6c")
	players[1].hand = cardsFromString("2h,3h,4h,5h,6h")
	game.trumpCard = cardFromString("14d")

	game.playerDidDiscard(players[1], []*deck.Card{})
	game.playerDidDiscard(players[0], []*deck.Card{})
	assert.NoError(t, game.replaceDiscards())

	cardFunc = createPlayCardFunc(t, game, []*Player{players[0], players[1]})
	cardFunc(1, 0)
	cardFunc(0, 0)
	game.nextRound()
	cardFunc(0, 0)
	cardFunc(1, 0)
	game.nextRound()
	cardFunc(1, 0)
	cardFunc(0, 0)
	game.nextRound()
	cardFunc(0, 0)
	cardFunc(1, 0)
	game.nextRound()
	cardFunc(1, 0)
	cardFunc(0, 0)
	game.nextRound()

	game.done = true
	details, over := game.GetEndOfGameDetails()

	assert.True(t, over)
	assert.NotNil(t, details)
	assert.Equal(t, 1, len(game.foldedPlayers))
	assert.Equal(t, 2, len(game.playerOrder))
	assert.Equal(t, []*Player{players[0]}, game.result.Booted)
	assert.Equal(t, []*Player{players[1]}, game.result.Winners)
	assert.Equal(t, []*Player{players[2]}, game.result.Folded)

	expect := map[int64]int{
		0: -50,
		1: 100,
		2: -50,
	}
	assert.Equal(t, expect, details.BalanceAdjustments)
}
