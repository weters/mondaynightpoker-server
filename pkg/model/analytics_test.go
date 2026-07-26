package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allTime is the widest range the date-filtered analytics queries accept, used by
// tests that are not exercising the filter itself.
func allTime() (from, to time.Time) {
	return time.Unix(0, 0), time.Now().Add(time.Hour)
}

// analyticsPlayer returns a site admin, which exempts the player from the
// create-another-table cool-down. These tests need several tables per player to
// have more than one night to analyze.
func analyticsPlayer(t *testing.T) *Player {
	t.Helper()

	p := player()
	p.IsSiteAdmin = true
	require.NoError(t, testRepos.Players.Save(cbg, p))

	return p
}

// endGame creates a game at the table and immediately ends it with the given
// per-player adjustments, which is the only path that writes game-driven ledger
// rows. The stored game_type is a real display name so it resolves to a parser.
func endGame(t *testing.T, tbl *Table, adjustments map[int64]int) *Game {
	t.Helper()

	g, err := testRepos.Games.CreateGame(cbg, tbl, "Bourré")
	require.NoError(t, err)
	require.NoError(t, testRepos.Games.EndGame(cbg, g, map[string]any{"ante": 50}, adjustments))

	return g
}

// adjustBalanceWithoutGame writes a ledger row with no game_id, the shape a manual
// balance correction takes. There is no repository method for it, so the test goes
// through the same stored procedure the game path uses.
func adjustBalanceWithoutGame(t *testing.T, pt *PlayerTable, amount int, reason string) {
	t.Helper()

	_, err := testDB.ExecContext(cbg, "SELECT adjust_balance($1, $2, $3, NULL, $4)", pt.ID, pt.Balance, amount, reason)
	require.NoError(t, err)
}

func TestNewSpread(t *testing.T) {
	// A ground-out result set and a swingy one with the same total: the whole point
	// of reporting a spread is that these are not the same player.
	steady := NewSpread([]int{10, 10, 10, 10})
	swingy := NewSpread([]int{500, -400, 300, -360})

	assert.Equal(t, 40, steady.TotalCents)
	assert.Equal(t, 40, swingy.TotalCents)
	assert.Equal(t, 10.0, steady.MeanCents)
	assert.Equal(t, 10.0, swingy.MeanCents)
	assert.Zero(t, steady.StdDevCents)
	assert.Greater(t, swingy.StdDevCents, 400.0)
}

// TestNewSpread_Values covers the field math across the shapes that behave
// differently: an odd count, an even count (median averages the middle pair), a
// lone result (deviation undefined, reported as zero rather than misleadingly
// small), and nothing at all.
func TestNewSpread_Values(t *testing.T) {
	testCases := []struct {
		name string
		in   []int
		want Spread
	}{
		{
			name: "odd count",
			in:   []int{100, -50, 25},
			want: Spread{Count: 3, TotalCents: 75, MeanCents: 25, MedianCents: 25, StdDevCents: 75, BestCents: 100, WorstCents: -50},
		},
		{
			name: "even count averages the middle pair",
			in:   []int{10, 20, 30, 40},
			want: Spread{Count: 4, TotalCents: 100, MeanCents: 25, MedianCents: 25, StdDevCents: 12.909944487358056, BestCents: 40, WorstCents: 10},
		},
		{
			name: "single result has no deviation",
			in:   []int{250},
			want: Spread{Count: 1, TotalCents: 250, MeanCents: 250, MedianCents: 250, BestCents: 250, WorstCents: 250},
		},
		{
			name: "empty",
			in:   nil,
			want: Spread{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewSpread(tc.in)

			assert.Equal(t, tc.want.Count, got.Count)
			assert.Equal(t, tc.want.TotalCents, got.TotalCents)
			assert.Equal(t, tc.want.BestCents, got.BestCents)
			assert.Equal(t, tc.want.WorstCents, got.WorstCents)
			assert.InDelta(t, tc.want.MeanCents, got.MeanCents, 0.001)
			assert.InDelta(t, tc.want.MedianCents, got.MedianCents, 0.001)
			assert.InDelta(t, tc.want.StdDevCents, got.StdDevCents, 0.001)
		})
	}
}

// streakInputs builds a chronological series one day apart, so ordering is
// unambiguous and the recorded timestamps are checkable.
func streakInputs(nets ...int) []StreakInput {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	inputs := make([]StreakInput, len(nets))
	for i, net := range nets {
		inputs[i] = StreakInput{NetCents: net, At: base.AddDate(0, 0, i)}
	}

	return inputs
}

func TestComputeStreaks(t *testing.T) {
	// Two wins, then three losses, then one win still running.
	got := ComputeStreaks(streakInputs(100, 50, -20, -30, -40, 75))

	assert.Equal(t, StreakWinning, got.LongestWinning.Outcome)
	assert.Equal(t, 2, got.LongestWinning.Length)
	assert.Equal(t, 150, got.LongestWinning.NetCents)

	assert.Equal(t, StreakLosing, got.LongestLosing.Outcome)
	assert.Equal(t, 3, got.LongestLosing.Length)
	assert.Equal(t, -90, got.LongestLosing.NetCents)

	assert.Equal(t, StreakWinning, got.Current.Outcome)
	assert.Equal(t, 1, got.Current.Length)
	assert.Equal(t, 75, got.Current.NetCents)
}

// TestComputeStreaks_BreakEvenBreaksRun pins the rule that a night where nothing
// changed is not a continued win.
func TestComputeStreaks_BreakEvenBreaksRun(t *testing.T) {
	got := ComputeStreaks(streakInputs(100, 100, 0, 100))

	assert.Equal(t, 2, got.LongestWinning.Length, "the zero result ended the run")

	// The series ends on a win, so a run is in progress; had it ended on the zero,
	// there would be no current streak at all.
	assert.Equal(t, StreakWinning, got.Current.Outcome)
	assert.Equal(t, 1, got.Current.Length)
}

func TestComputeStreaks_EndsOnBreakEven(t *testing.T) {
	got := ComputeStreaks(streakInputs(100, 100, 0))

	assert.Equal(t, 2, got.LongestWinning.Length)
	assert.Equal(t, StreakNone, got.Current.Outcome, "breaking even leaves no streak running")
	assert.Zero(t, got.Current.Length)
}

// TestComputeStreaks_TiePrefersEarlier pins tie-breaking: a later run has to be
// strictly longer to displace an earlier one.
func TestComputeStreaks_TiePrefersEarlier(t *testing.T) {
	got := ComputeStreaks(streakInputs(100, 200, -10, 300, 400))

	assert.Equal(t, 2, got.LongestWinning.Length)
	assert.Equal(t, 300, got.LongestWinning.NetCents, "the first of the two equal runs wins the tie")
}

func TestComputeStreaks_Timestamps(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := ComputeStreaks(streakInputs(100, 50, -20))

	assert.Equal(t, base, got.LongestWinning.StartedAt)
	assert.Equal(t, base.AddDate(0, 0, 1), got.LongestWinning.EndedAt)
	assert.Equal(t, base.AddDate(0, 0, 2), got.LongestLosing.StartedAt)
}

func TestComputeStreaks_Empty(t *testing.T) {
	got := ComputeStreaks(nil)

	assert.Equal(t, StreakNone, got.LongestWinning.Outcome)
	assert.Equal(t, StreakNone, got.LongestLosing.Outcome)
	assert.Equal(t, StreakNone, got.Current.Outcome)
}

func TestComputeStreaks_AllLosses(t *testing.T) {
	got := ComputeStreaks(streakInputs(-10, -20, -30))

	assert.Equal(t, 3, got.LongestLosing.Length)
	assert.Equal(t, -60, got.LongestLosing.NetCents)
	assert.Zero(t, got.LongestWinning.Length)
	assert.Equal(t, StreakLosing, got.Current.Outcome)
}

func TestPlayerRepo_GetPlayerVariance(t *testing.T) {
	p := analyticsPlayer(t)
	tbl1, err := testRepos.Tables.CreateTable(cbg, p, "night one")
	require.NoError(t, err)
	tbl2, err := testRepos.Tables.CreateTable(cbg, p, "night two")
	require.NoError(t, err)

	// Night one: up 100, up 50. Night two: down 30.
	endGame(t, tbl1, map[int64]int{p.ID: 100})
	endGame(t, tbl1, map[int64]int{p.ID: 50})
	endGame(t, tbl2, map[int64]int{p.ID: -30})

	from, to := allTime()
	got, err := testRepos.Players.GetPlayerVariance(cbg, p.ID, from, to)
	require.NoError(t, err)

	assert.Equal(t, 3, got.ByGame.Count)
	assert.Equal(t, 120, got.ByGame.TotalCents)
	assert.Equal(t, 100, got.ByGame.BestCents)
	assert.Equal(t, -30, got.ByGame.WorstCents)

	// Two nights, not three games: the session view aggregates each table.
	assert.Equal(t, 2, got.BySession.Count)
	assert.Equal(t, 120, got.BySession.TotalCents)
	assert.Equal(t, 150, got.BySession.BestCents)
	assert.Equal(t, -30, got.BySession.WorstCents)

	assert.Equal(t, 2, got.GameStreaks.LongestWinning.Length)
	assert.Equal(t, 1, got.SessionStreaks.LongestWinning.Length)
	assert.Equal(t, 1, got.SessionStreaks.LongestLosing.Length)
}

// TestPlayerRepo_GetPlayerVariance_ExcludesNonGameLedger pins that manual balance
// adjustments do not look like winning games.
func TestPlayerRepo_GetPlayerVariance_ExcludesNonGameLedger(t *testing.T) {
	p := analyticsPlayer(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, p, "adjustments")
	require.NoError(t, err)

	players, err := testRepos.Tables.GetPlayers(cbg, tbl)
	require.NoError(t, err)
	require.Len(t, players, 1, "creating a table seats its creator")
	adjustBalanceWithoutGame(t, players[0], 5000, "buy-in")

	endGame(t, tbl, map[int64]int{p.ID: -100})

	from, to := allTime()
	got, err := testRepos.Players.GetPlayerVariance(cbg, p.ID, from, to)
	require.NoError(t, err)

	assert.Equal(t, 1, got.ByGame.Count, "the buy-in is not a game")
	assert.Equal(t, -100, got.ByGame.TotalCents)
	assert.Equal(t, -100, got.BySession.TotalCents)
}

// TestPlayerRepo_GetPlayerVariance_SkipsTablesWithNoGames pins that a night spent
// watching does not appear as a break-even session and end a winning run.
func TestPlayerRepo_GetPlayerVariance_SkipsTablesWithNoGames(t *testing.T) {
	p := analyticsPlayer(t)
	played, err := testRepos.Tables.CreateTable(cbg, p, "played")
	require.NoError(t, err)
	// Creating a table seats its creator, so this is a night joined but never
	// played at.
	_, err = testRepos.Tables.CreateTable(cbg, p, "watched")
	require.NoError(t, err)

	endGame(t, played, map[int64]int{p.ID: 100})

	from, to := allTime()
	got, err := testRepos.Players.GetPlayerVariance(cbg, p.ID, from, to)
	require.NoError(t, err)

	assert.Equal(t, 1, got.BySession.Count)
	assert.Equal(t, StreakWinning, got.SessionStreaks.Current.Outcome)
}

func TestPlayerRepo_GetPlayerVariance_NoResults(t *testing.T) {
	p := analyticsPlayer(t)

	from, to := allTime()
	got, err := testRepos.Players.GetPlayerVariance(cbg, p.ID, from, to)
	require.NoError(t, err)

	assert.Zero(t, got.ByGame.Count)
	assert.Zero(t, got.BySession.Count)
	assert.Equal(t, StreakNone, got.SessionStreaks.Current.Outcome)
}

func TestPlayerRepo_GetPlayerGameLogs(t *testing.T) {
	p := analyticsPlayer(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, p, "logged")
	require.NoError(t, err)

	// A finished game with a log, and one that never finished.
	withLog, err := testRepos.Games.CreateGame(cbg, tbl, "Bourré")
	require.NoError(t, err)
	require.NoError(t, testRepos.Games.EndGame(cbg, withLog, map[string]any{"ante": 50}, map[int64]int{p.ID: 100}))

	_, err = testRepos.Games.CreateGame(cbg, tbl, "Bourré")
	require.NoError(t, err)

	from, to := allTime()
	logs, total, err := testRepos.Players.GetPlayerGameLogs(cbg, p.ID, from, to, 100)
	require.NoError(t, err)

	require.Len(t, logs, 1, "an unfinished game has no log to analyze")
	assert.Equal(t, int64(1), total)
	assert.Equal(t, withLog.ID, logs[0].GameID)
	assert.Equal(t, tbl.UUID, logs[0].TableUUID)
	assert.Equal(t, "Bourré", logs[0].GameType)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(logs[0].Data, &payload))
	assert.Equal(t, float64(50), payload["ante"])
}

// TestPlayerRepo_GetPlayerGameLogs_Limit pins that the count reports the full
// match set even when the returned rows are capped, so a caller can say its
// analysis was partial rather than silently truncating.
func TestPlayerRepo_GetPlayerGameLogs_Limit(t *testing.T) {
	p := analyticsPlayer(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, p, "many games")
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		g, err := testRepos.Games.CreateGame(cbg, tbl, "Bourré")
		require.NoError(t, err)
		require.NoError(t, testRepos.Games.EndGame(cbg, g, map[string]any{"ante": 50}, map[int64]int{p.ID: 10}))
	}

	from, to := allTime()
	// The window-function total reports the full match set even though the page is
	// capped, which is what lets a caller say its analysis was partial.
	logs, total, err := testRepos.Players.GetPlayerGameLogs(cbg, p.ID, from, to, 2)
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, int64(3), total)
}

// TestPlayerRepo_GetPlayerGameLogs_ExcludesOtherPlayers pins that participation is
// defined by the ledger: a game at a table the player is at, but did not play in,
// is not theirs.
func TestPlayerRepo_GetPlayerGameLogs_ExcludesOtherPlayers(t *testing.T) {
	p1 := analyticsPlayer(t)
	p2 := analyticsPlayer(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, p1, "shared")
	require.NoError(t, err)

	_, err = testRepos.Tables.Join(cbg, p2, tbl)
	require.NoError(t, err)

	g, err := testRepos.Games.CreateGame(cbg, tbl, "Bourré")
	require.NoError(t, err)
	require.NoError(t, testRepos.Games.EndGame(cbg, g, map[string]any{"ante": 50}, map[int64]int{p1.ID: 100}))

	from, to := allTime()

	logs, _, err := testRepos.Players.GetPlayerGameLogs(cbg, p1.ID, from, to, 100)
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	logs, total, err := testRepos.Players.GetPlayerGameLogs(cbg, p2.ID, from, to, 100)
	require.NoError(t, err)
	assert.Empty(t, logs, "sitting at the table is not playing the game")
	assert.Zero(t, total, "no rows means no window-function total")
}

func TestPlayerRepo_GetPlayerGameLogs_ExcludesDeletedTables(t *testing.T) {
	p := analyticsPlayer(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, p, "doomed")
	require.NoError(t, err)

	g, err := testRepos.Games.CreateGame(cbg, tbl, "Bourré")
	require.NoError(t, err)
	require.NoError(t, testRepos.Games.EndGame(cbg, g, map[string]any{"ante": 50}, map[int64]int{p.ID: 100}))

	tbl.Deleted = true
	require.NoError(t, testRepos.Tables.Save(cbg, tbl))

	from, to := allTime()

	logs, _, err := testRepos.Players.GetPlayerGameLogs(cbg, p.ID, from, to, 100)
	require.NoError(t, err)
	assert.Empty(t, logs)

	got, err := testRepos.Players.GetPlayerVariance(cbg, p.ID, from, to)
	require.NoError(t, err)
	assert.Zero(t, got.ByGame.Count, "a deleted table is invisible to every read")
}
