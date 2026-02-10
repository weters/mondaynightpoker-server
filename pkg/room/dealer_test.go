package room

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/model"
)

func TestDealer_AddClient(t *testing.T) {
	d := NewDealer(&PitBoss{}, &model.Table{})
	c := NewClient(nil, nil, nil)
	c2 := NewClient(nil, nil, nil)

	d.AddClient(c)
	d.AddClient(c2)

	assert.False(t, d.RemoveClient(c))
	assert.True(t, d.RemoveClient(c2))
}

func newPlayerTable(id int64, displayName string, active bool, blocked bool) *model.PlayerTable {
	return &model.PlayerTable{
		Player: &model.Player{
			ID:          id,
			DisplayName: displayName,
		},
		PlayerID:  id,
		Active:    active,
		IsBlocked: blocked,
	}
}

func Test_buildPickerLogMessage(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Charlie", true, false),
		newPlayerTable(2, "Alice", true, false),
		newPlayerTable(3, "Bob", true, false),
	}

	// Alice (2) picked, sorted order: Alice, Bob, Charlie => next is Bob
	msgs := buildPickerLogMessage(players, 2)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "Alice picked the last game. Bob is next to pick", msgs[0].Message)
	assert.NotEmpty(t, msgs[0].UUID)

	// Bob (3) picked, sorted order: Alice, Bob, Charlie => next is Charlie
	msgs = buildPickerLogMessage(players, 3)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "Bob picked the last game. Charlie is next to pick", msgs[0].Message)

	// Charlie (1) picked, sorted order: Alice, Bob, Charlie => next wraps to Alice
	msgs = buildPickerLogMessage(players, 1)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "Charlie picked the last game. Alice is next to pick", msgs[0].Message)
}

func Test_buildPickerLogMessage_singlePlayer(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Alice", true, false),
	}

	msgs := buildPickerLogMessage(players, 1)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "Alice picked the last game. Alice is next to pick", msgs[0].Message)
}

func Test_buildPickerLogMessage_pickerNotFound(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Alice", true, false),
		newPlayerTable(2, "Bob", true, false),
	}

	msgs := buildPickerLogMessage(players, 99)
	assert.Nil(t, msgs)
}

func Test_buildPickerLogMessage_noActivePlayers(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Alice", false, false),
		newPlayerTable(2, "Bob", true, true), // blocked
	}

	msgs := buildPickerLogMessage(players, 1)
	assert.Nil(t, msgs)
}

func Test_buildPickerLogMessage_emptyPlayers(t *testing.T) {
	msgs := buildPickerLogMessage([]*model.PlayerTable{}, 1)
	assert.Nil(t, msgs)
}

func Test_buildPickerLogMessage_skipsInactivePlayers(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Alice", true, false),
		newPlayerTable(2, "Bob", false, false),   // inactive
		newPlayerTable(3, "Charlie", true, true), // blocked
		newPlayerTable(4, "Darren", true, false),
	}

	// Sorted active: Alice, Darren. Alice picked => next is Darren
	msgs := buildPickerLogMessage(players, 1)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "Alice picked the last game. Darren is next to pick", msgs[0].Message)

	// Darren picked => next wraps to Alice
	msgs = buildPickerLogMessage(players, 4)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "Darren picked the last game. Alice is next to pick", msgs[0].Message)
}
