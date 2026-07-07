package room

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
)

// fakeStore is a programmable TableStore for exercising the dealer's
// concurrency behavior without a database
type fakeStore struct {
	mu sync.Mutex

	players         []*model.PlayerTable
	rosters         [][]*model.PlayerTable // per-call GetPlayers overrides
	getPlayersDelay []time.Duration        // per-call delays; last value repeats
	getPlayersCalls int

	createGameErrs  []error // per-call results; nil-padded
	createGameCalls int
	endGameErrs     []error
	endGameDelay    time.Duration
	endGameCalls    int

	saved []*model.PlayerTable
}

func (f *fakeStore) GetPlayerByID(_ context.Context, id int64) (*model.Player, error) {
	return &model.Player{ID: id}, nil
}

func (f *fakeStore) GetPlayerTable(_ context.Context, player *model.Player, _ *model.Table) (*model.PlayerTable, error) {
	return &model.PlayerTable{Player: player, PlayerID: player.ID, Active: true}, nil
}

func (f *fakeStore) SavePlayerTable(_ context.Context, pt *model.PlayerTable) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, pt)
	return nil
}

func (f *fakeStore) GetPlayers(_ context.Context, _ *model.Table) ([]*model.PlayerTable, error) {
	f.mu.Lock()
	call := f.getPlayersCalls
	f.getPlayersCalls++

	var delay time.Duration
	if n := len(f.getPlayersDelay); n > 0 {
		if call < n {
			delay = f.getPlayersDelay[call]
		} else {
			delay = f.getPlayersDelay[n-1]
		}
	}

	players := f.players
	if call < len(f.rosters) {
		players = f.rosters[call]
	}
	f.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	return players, nil
}

func (f *fakeStore) GetActivePlayersShifted(_ context.Context, _ *model.Table) ([]*model.PlayerTable, error) {
	f.mu.Lock()
	players := f.players
	f.mu.Unlock()
	return players, nil
}

func (f *fakeStore) CreateGame(_ context.Context, _ *model.Table, _ string) (*model.Game, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	call := f.createGameCalls
	f.createGameCalls++
	if call < len(f.createGameErrs) && f.createGameErrs[call] != nil {
		return nil, f.createGameErrs[call]
	}

	return &model.Game{ID: int64(call + 1)}, nil
}

func (f *fakeStore) EndGame(_ context.Context, _ *model.Game, _ interface{}, _ map[int64]int) error {
	f.mu.Lock()
	call := f.endGameCalls
	f.endGameCalls++
	delay := f.endGameDelay
	var err error
	if call < len(f.endGameErrs) {
		err = f.endGameErrs[call]
	}
	f.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	return err
}

func (f *fakeStore) counts() (getPlayers, createGame, endGame int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getPlayersCalls, f.createGameCalls, f.endGameCalls
}

// fakeGame is a minimal playable.Playable
type fakeGame struct {
	*playable.Core
}

func newFakeGame() *fakeGame {
	return &fakeGame{Core: playable.NewCore()}
}

func (g *fakeGame) Action(int64, *playable.PayloadIn) (*playable.Response, bool, error) {
	return nil, false, nil
}

func (g *fakeGame) GetPlayerState(int64) (*playable.Response, error) {
	return &playable.Response{Key: "game", Value: "fake"}, nil
}

func (g *fakeGame) GetEndOfGameDetails() (*playable.GameOverDetails, bool) {
	return nil, false
}

func (g *fakeGame) Name() string {
	return "fake"
}

func fakePlayerTable(id int64) *model.PlayerTable {
	return &model.PlayerTable{
		Player:     &model.Player{ID: id},
		PlayerID:   id,
		Active:     true,
		TableStake: 10000,
	}
}

func newFakeDealer(store *fakeStore) *Dealer {
	pb := NewPitBoss(store, PitBossOptions{StartGameDelay: time.Millisecond})
	return NewDealer(pb, &model.Table{UUID: "fake-table"})
}

// probe waits until the run loop executes a no-op, proving it is responsive
func probeRunLoop(t *testing.T, d *Dealer, within time.Duration) {
	t.Helper()
	done := make(chan struct{})
	d.execInRunLoop <- func() { close(done) }
	select {
	case <-done:
	case <-time.After(within):
		t.Fatalf("run loop did not respond within %s", within)
	}
}

func TestDealer_slowRosterFetchDoesNotBlockRunLoop(t *testing.T) {
	store := &fakeStore{
		players:         []*model.PlayerTable{fakePlayerTable(1)},
		getPlayersDelay: []time.Duration{500 * time.Millisecond},
	}
	d := newFakeDealer(store)
	d.StartShift()
	defer d.EndShift()

	c := NewClient(nil, &model.Player{ID: 1}, &model.Table{})
	d.AddClient(c) // triggers a roster fetch that sleeps 500ms

	start := time.Now()
	probeRunLoop(t, d, 200*time.Millisecond)
	assert.Less(t, time.Since(start), 200*time.Millisecond, "run loop must not wait on the roster fetch")
}

func TestDealer_staleRosterDropped(t *testing.T) {
	store := &fakeStore{
		rosters: [][]*model.PlayerTable{
			{fakePlayerTable(1)},                     // AddClient fetch
			{fakePlayerTable(1)},                     // slow, stale fetch
			{fakePlayerTable(1), fakePlayerTable(2)}, // fast, current fetch
		},
		getPlayersDelay: []time.Duration{0, 300 * time.Millisecond, 10 * time.Millisecond},
	}
	d := newFakeDealer(store)
	d.StartShift()
	defer d.EndShift()

	c := NewClient(nil, &model.Player{ID: 1}, &model.Table{})
	d.AddClient(c)

	// wait for the initial clientState from AddClient
	waitForClientState(t, c, time.Second)

	// request a roster and wait until its (slow) fetch is in flight...
	d.stateChanged <- stateClientEvent
	require.Eventually(t, func() bool {
		getPlayers, _, _ := store.counts()
		return getPlayers == 2
	}, time.Second, 5*time.Millisecond)

	// ...then request a newer one whose fetch completes first
	d.stateChanged <- stateClientEvent

	// only the newest fetch may be delivered
	first := waitForClientState(t, c, time.Second)
	assert.Len(t, first, 2, "the delivered roster must be from the newest fetch")

	select {
	case msg := <-c.SendChan():
		if resp, ok := msg.(playable.Response); ok && resp.Key == "clientState" {
			t.Fatal("the stale roster must be dropped, but a second clientState arrived")
		}
	case <-time.After(500 * time.Millisecond):
	}
}

func waitForClientState(t *testing.T, c *Client, within time.Duration) map[int64]*clientStatePlayers {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case msg := <-c.SendChan():
			if resp, ok := msg.(playable.Response); ok && resp.Key == "clientState" {
				cs, ok := resp.Data.(map[int64]*clientStatePlayers)
				require.True(t, ok)
				return cs
			}
		case <-deadline:
			t.Fatalf("did not receive clientState within %s", within)
			return nil
		}
	}
}

func TestDealer_endGamePersistRetries(t *testing.T) {
	failure := assert.AnError
	store := &fakeStore{
		players:        []*model.PlayerTable{fakePlayerTable(1), fakePlayerTable(2)},
		createGameErrs: []error{failure, failure, nil},
	}
	d := newFakeDealer(store)
	d.persistRetryDelays = []time.Duration{0, 0}
	d.StartShift()
	defer d.EndShift()

	d.execInRunLoop <- func() {
		d.game = newFakeGame()
		d.endGame(d.game, &playable.GameOverDetails{BalanceAdjustments: map[int64]int{1: 100, 2: -100}})
	}

	require.Eventually(t, func() bool {
		_, createGame, endGame := store.counts()
		return createGame == 3 && endGame == 1
	}, 2*time.Second, 10*time.Millisecond, "persistence must retry until it succeeds")

	// and the persisting gate must be released
	require.Eventually(t, func() bool {
		result := make(chan bool, 1)
		d.execInRunLoop <- func() { result <- !d.persistingGame }
		return <-result
	}, 2*time.Second, 10*time.Millisecond)
}

func TestDealer_deferredStartWaitsForPersist(t *testing.T) {
	store := &fakeStore{
		players:      []*model.PlayerTable{fakePlayerTable(1), fakePlayerTable(2), fakePlayerTable(3)},
		endGameDelay: 300 * time.Millisecond,
	}
	d := newFakeDealer(store)
	d.persistRetryDelays = []time.Duration{0, 0}
	d.StartShift()
	defer d.EndShift()

	// finish a game; its results persist slowly in the background
	d.execInRunLoop <- func() {
		d.game = newFakeGame()
		d.endGame(d.game, &playable.GameOverDetails{})
	}

	// immediately schedule the next game (1ms start delay)
	admin := NewClient(nil, &model.Player{ID: 1, IsSiteAdmin: true}, &model.Table{UUID: "fake-table"})
	d.ReceivedMessage(admin, &playable.PayloadIn{
		Action:  "createGame",
		Subject: "guts",
		AdditionalData: playable.AdditionalData{
			"ante": float64(25),
		},
	})

	// while persistence is still running, the start must be deferred
	time.Sleep(100 * time.Millisecond)
	deferred := make(chan bool, 1)
	d.execInRunLoop <- func() { deferred <- d.deferredStart != nil && d.game == nil }
	assert.True(t, <-deferred, "the game start must wait for persistence to finish")

	// once persistence completes, the deferred game must start
	require.Eventually(t, func() bool {
		started := make(chan bool, 1)
		d.execInRunLoop <- func() { started <- d.game != nil }
		return <-started
	}, 3*time.Second, 20*time.Millisecond, "the deferred game must start after persistence completes")
}
