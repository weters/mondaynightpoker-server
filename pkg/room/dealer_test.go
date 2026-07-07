package room

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/model"
)

func TestDealer_AddClient(t *testing.T) {
	d := NewDealer(NewPitBoss(testStore, PitBossOptions{StartGameDelay: time.Second}), &model.Table{})
	d.StartShift()
	defer d.EndShift()

	c := NewClient(nil, &model.Player{ID: 1}, &model.Table{})
	c2 := NewClient(nil, &model.Player{ID: 2}, &model.Table{})

	d.AddClient(c)
	d.AddClient(c2)

	assert.False(t, d.RemoveClient(c))
	assert.True(t, d.RemoveClient(c2))
}

// TestDealer_clientChurnRace exercises concurrent client adds/removes while the run
// loop iterates the clients map. Run with -race; fails if the map is mutated outside
// the run loop.
func TestDealer_clientChurnRace(t *testing.T) {
	d := NewDealer(NewPitBoss(testStore, PitBossOptions{StartGameDelay: time.Second}), &model.Table{})
	d.StartShift()

	done := make(chan bool)
	pumpStopped := make(chan bool)
	go func() {
		defer close(pumpStopped)
		for {
			select {
			case <-done:
				return
			default:
				// sendGameScheduled ranges the clients map without touching the database
				d.stateChanged <- stateGameScheduled
			}
		}
	}()

	var wg sync.WaitGroup
	for i := int64(1); i <= 50; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			c := NewClient(nil, &model.Player{ID: id}, &model.Table{})
			d.AddClient(c)
			d.RemoveClient(c)
		}(i)
	}

	wg.Wait()
	close(done)
	<-pumpStopped

	// every add was paired with a remove, so one final client must be the last one
	c := NewClient(nil, &model.Player{ID: 999}, &model.Table{})
	d.AddClient(c)
	assert.True(t, d.RemoveClient(c))

	d.EndShift()
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

func Test_getNextPicker(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Charlie", true, false),
		newPlayerTable(2, "Alice", true, false),
		newPlayerTable(3, "Bob", true, false),
	}

	// Sorted: Alice, Bob, Charlie
	next := getNextPicker(players, 2) // Alice picked => Bob next
	assert.NotNil(t, next)
	assert.Equal(t, int64(3), next.PlayerID)

	next = getNextPicker(players, 3) // Bob picked => Charlie next
	assert.NotNil(t, next)
	assert.Equal(t, int64(1), next.PlayerID)

	next = getNextPicker(players, 1) // Charlie picked => wraps to Alice
	assert.NotNil(t, next)
	assert.Equal(t, int64(2), next.PlayerID)
}

func Test_getNextPicker_pickerNotFound(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Alice", true, false),
		newPlayerTable(2, "Bob", true, false),
	}

	assert.Nil(t, getNextPicker(players, 99))
}

func Test_getNextPicker_noActivePlayers(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Alice", false, false),
		newPlayerTable(2, "Bob", true, true),
	}

	assert.Nil(t, getNextPicker(players, 1))
}

func Test_getNextPicker_singlePlayerReturnsSelf(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Alice", true, false),
	}

	next := getNextPicker(players, 1)
	assert.NotNil(t, next)
	assert.Equal(t, int64(1), next.PlayerID)
}

func Test_getNextPicker_skipsInactiveAndBlocked(t *testing.T) {
	players := []*model.PlayerTable{
		newPlayerTable(1, "Alice", true, false),
		newPlayerTable(2, "Bob", false, false),   // inactive
		newPlayerTable(3, "Charlie", true, true), // blocked
		newPlayerTable(4, "Darren", true, false),
	}

	// Active sorted: Alice, Darren. Alice picked => Darren next.
	next := getNextPicker(players, 1)
	assert.NotNil(t, next)
	assert.Equal(t, int64(4), next.PlayerID)

	// Darren picked => wraps to Alice
	next = getNextPicker(players, 4)
	assert.NotNil(t, next)
	assert.Equal(t, int64(1), next.PlayerID)

	// Previous picker is no longer active => no next picker
	assert.Nil(t, getNextPicker(players, 2))
	assert.Nil(t, getNextPicker(players, 3))
}
