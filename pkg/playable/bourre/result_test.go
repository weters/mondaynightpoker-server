package bourre

import (
	"github.com/stretchr/testify/assert"
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
