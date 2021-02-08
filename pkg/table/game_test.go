package table

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGame_EndGame(t *testing.T) {
	player, table, game := playerTableAndGame()

	g2, err := GameByID(cbg, game.ID)
	assert.NoError(t, err)
	assert.NotNil(t, g2)
	assert.Nil(t, g2.data)
	assert.True(t, g2.Ended.IsZero())

	playerTable, err := player.GetPlayerTable(cbg, table)
	assert.NoError(t, err)

	before := time.Now()
	err = game.EndGame(cbg, map[string]string{"foo": "bar", "tar": "car"}, map[int64]int{playerTable.PlayerID: 123})
	assert.NoError(t, err)

	playerTable, _ = player.GetPlayerTable(cbg, table)
	assert.Equal(t, 123, playerTable.Balance)

	g2, err = GameByID(cbg, game.ID)
	assert.NoError(t, err)
	assert.NotNil(t, g2)
	assert.Equal(t, "bar", g2.data.(map[string]interface{})["foo"])
	assert.True(t, g2.Ended.After(before))
}

func playerTableAndGame() (*Player, *Table, *Game) {
	p, tbl := playerAndTable()
	game, err := tbl.CreateGame(cbg, "bourre")
	if err != nil {
		panic(err)
	}

	return p, tbl, game
}
