package model

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"time"
)

// GameLogRecord is one completed game's persisted log plus the identity needed to
// interpret it. GameType holds the games.game_type display name, which the caller
// resolves to a parser.
type GameLogRecord struct {
	GameID    int64
	TableUUID string
	TableName string
	GameType  string
	Created   time.Time
	Data      json.RawMessage
}

// GetPlayerGameLogs returns the persisted logs of completed games the player took
// part in, newest first, capped at limit rows. The second return value is how many
// games matched in total, ignoring the cap, so a caller can report that its
// analysis covered only part of the range rather than silently truncating.
//
// The total comes from a window function rather than a second query: window
// functions are evaluated before LIMIT, so one statement yields both the page and
// the full count from a single consistent snapshot. A separate count query would
// have to repeat this predicate verbatim, and the two would drift.
//
// Games with no log are excluded: a game row is created when the game starts and
// only receives its data payload when it ends, so an in-progress or terminated
// game has nothing to analyze. Participation is defined by the ledger — the player
// has a transaction against the game — which is the same definition GetPlayerStats
// uses for games played.
//
// The date range filters on tables.created rather than the game's own timestamp,
// matching GetPlayerStats: a table is a single night by convention, so normalizing
// every figure onto the table's date keeps the analytics tools reconcilable with
// each other.
func (r *PlayerRepo) GetPlayerGameLogs(ctx context.Context, playerID int64, from, to time.Time, limit int) (records []*GameLogRecord, total int64, err error) {
	const query = `
SELECT g.id, t.uuid, t.name, g.game_type, g.created, g.data, COUNT(*) OVER () AS total
FROM games g
INNER JOIN tables t ON g.table_uuid = t.uuid
WHERE NOT t.deleted
  AND g.data IS NOT NULL AND jsonb_typeof(g.data) <> 'null'
  AND t.created >= $2 AND t.created <= $3
  AND EXISTS (
    SELECT 1
    FROM players_tables_transactions ptt
    INNER JOIN players_tables pt ON ptt.players_tables_id = pt.id
    WHERE ptt.game_id = g.id
      AND pt.player_id = $1
  )
ORDER BY g.created DESC, g.id DESC
LIMIT $4`

	rows, err := r.db.QueryContext(ctx, query, playerID, from, to, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	records = make([]*GameLogRecord, 0)
	for rows.Next() {
		var rec GameLogRecord
		if err := rows.Scan(&rec.GameID, &rec.TableUUID, &rec.TableName, &rec.GameType, &rec.Created, &rec.Data, &total); err != nil {
			return nil, 0, err
		}

		records = append(records, &rec)
	}

	return records, total, rows.Err()
}

// PlayerGameResult is one game's net outcome for a player, in cents.
type PlayerGameResult struct {
	GameID    int64
	TableUUID string
	GameType  string
	Created   time.Time
	NetCents  int
}

// PlayerSessionResult is one night's net outcome for a player, in cents. A session
// is a table: the site's convention is one table per night.
type PlayerSessionResult struct {
	TableUUID   string
	TableName   string
	Created     time.Time
	NetCents    int
	GamesPlayed int
}

// GetPlayerGameResults returns the player's per-game net results, oldest first.
//
// Ordering is chronological because these results feed streak detection, which is
// meaningless out of order. Only game-driven ledger entries count; manual balance
// adjustments have no game_id and are excluded, so a buy-in correction cannot look
// like a winning game.
func (r *PlayerRepo) GetPlayerGameResults(ctx context.Context, playerID int64, from, to time.Time) ([]*PlayerGameResult, error) {
	const query = `
SELECT g.id, t.uuid, g.game_type, g.created, SUM(ptt.adjustment)
FROM players_tables_transactions ptt
INNER JOIN players_tables pt ON ptt.players_tables_id = pt.id
INNER JOIN games g ON ptt.game_id = g.id
INNER JOIN tables t ON pt.table_uuid = t.uuid
WHERE pt.player_id = $1
  AND NOT t.deleted
  AND t.created >= $2 AND t.created <= $3
GROUP BY g.id, t.uuid, g.game_type, g.created
ORDER BY g.created ASC, g.id ASC`

	rows, err := r.db.QueryContext(ctx, query, playerID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*PlayerGameResult, 0)
	for rows.Next() {
		var res PlayerGameResult
		if err := rows.Scan(&res.GameID, &res.TableUUID, &res.GameType, &res.Created, &res.NetCents); err != nil {
			return nil, err
		}

		results = append(results, &res)
	}

	return results, rows.Err()
}

// GetPlayerSessionResults returns the player's per-night net results, oldest first.
//
// Tables the player joined but never played a game at are excluded. Including them
// would inject break-even nights that never happened, and since a break-even
// result breaks a streak, a night spent watching would end a winning run.
func (r *PlayerRepo) GetPlayerSessionResults(ctx context.Context, playerID int64, from, to time.Time) ([]*PlayerSessionResult, error) {
	const query = `
SELECT t.uuid,
       t.name,
       t.created,
       COALESCE(SUM(ptt.adjustment) FILTER (WHERE ptt.game_id IS NOT NULL), 0) AS net,
       COUNT(DISTINCT ptt.game_id) FILTER (WHERE ptt.game_id IS NOT NULL) AS games_played
FROM players_tables pt
INNER JOIN tables t ON pt.table_uuid = t.uuid
LEFT JOIN players_tables_transactions ptt ON ptt.players_tables_id = pt.id
WHERE pt.player_id = $1
  AND NOT t.deleted
  AND t.created >= $2 AND t.created <= $3
GROUP BY t.uuid, t.name, t.created
HAVING COUNT(DISTINCT ptt.game_id) FILTER (WHERE ptt.game_id IS NOT NULL) > 0
ORDER BY t.created ASC, t.uuid ASC`

	rows, err := r.db.QueryContext(ctx, query, playerID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*PlayerSessionResult, 0)
	for rows.Next() {
		var res PlayerSessionResult
		if err := rows.Scan(&res.TableUUID, &res.TableName, &res.Created, &res.NetCents, &res.GamesPlayed); err != nil {
			return nil, err
		}

		results = append(results, &res)
	}

	return results, rows.Err()
}

// Spread describes the distribution of a series of results, in cents.
//
// It exists because a total tells you nothing about how it was earned: two players
// up $50 look identical until you see that one ground it out and the other swung
// $400 in both directions. StdDevCents is the sample standard deviation and is
// zero for fewer than two results, where it is undefined rather than small.
type Spread struct {
	Count       int
	TotalCents  int
	MeanCents   float64
	StdDevCents float64
	MedianCents float64
	BestCents   int
	WorstCents  int
}

// NewSpread computes the distribution of the given results.
func NewSpread(values []int) Spread {
	s := Spread{Count: len(values)}
	if len(values) == 0 {
		return s
	}

	for _, v := range values {
		s.TotalCents += v
	}

	s.MeanCents = float64(s.TotalCents) / float64(len(values))

	if len(values) > 1 {
		var sumSquares float64
		for _, v := range values {
			d := float64(v) - s.MeanCents
			sumSquares += d * d
		}

		s.StdDevCents = math.Sqrt(sumSquares / float64(len(values)-1))
	}

	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

	// The median needs the values ordered anyway, so the extremes come from the
	// ends of the sorted copy rather than from a second pass with two branches.
	s.WorstCents = sorted[0]
	s.BestCents = sorted[len(sorted)-1]

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		s.MedianCents = float64(sorted[mid-1]+sorted[mid]) / 2
	} else {
		s.MedianCents = float64(sorted[mid])
	}

	return s
}

// StreakOutcome describes what a run of results has in common.
type StreakOutcome string

// Streak outcomes.
const (
	StreakNone    StreakOutcome = ""
	StreakWinning StreakOutcome = "winning"
	StreakLosing  StreakOutcome = "losing"
)

// Streak is an unbroken run of results with the same outcome.
//
// A break-even result belongs to neither run and ends whatever streak was in
// progress: a night where nothing changed is not a continued win.
type Streak struct {
	Outcome   StreakOutcome
	Length    int
	NetCents  int
	StartedAt time.Time
	EndedAt   time.Time
}

// Streaks summarizes the runs within a chronological series of results.
type Streaks struct {
	LongestWinning Streak
	LongestLosing  Streak
	// Current is the run still in progress as of the most recent result. It is the
	// zero Streak when the latest result broke even.
	Current Streak
}

// StreakInput is one dated result feeding streak detection.
type StreakInput struct {
	NetCents int
	At       time.Time
}

// ComputeStreaks finds the longest winning and losing runs in a chronological
// series, plus the run still in progress.
//
// Results must be ordered oldest first; the repository queries that feed this
// order that way for exactly this reason. Ties are resolved in favor of the
// earlier run, so a later streak has to actually be longer to displace it.
func ComputeStreaks(results []StreakInput) Streaks {
	var streaks Streaks
	var current Streak

	flush := func() {
		switch current.Outcome {
		case StreakWinning:
			if current.Length > streaks.LongestWinning.Length {
				streaks.LongestWinning = current
			}
		case StreakLosing:
			if current.Length > streaks.LongestLosing.Length {
				streaks.LongestLosing = current
			}
		}
	}

	for _, res := range results {
		outcome := StreakNone
		switch {
		case res.NetCents > 0:
			outcome = StreakWinning
		case res.NetCents < 0:
			outcome = StreakLosing
		}

		if outcome != current.Outcome {
			flush()
			current = Streak{Outcome: outcome, StartedAt: res.At}
		}

		current.Length++
		current.NetCents += res.NetCents
		current.EndedAt = res.At
	}

	flush()

	if current.Outcome != StreakNone {
		streaks.Current = current
	}

	return streaks
}

// PlayerVariance is a player's consistency profile: how their results are spread
// out and how they ran in streaks, at both game and night granularity.
type PlayerVariance struct {
	ByGame    Spread
	BySession Spread

	GameStreaks    Streaks
	SessionStreaks Streaks
}

// GetPlayerVariance computes a player's spread and streaks over the given range.
//
// Both granularities are reported because they answer different questions. The
// per-game spread is the honest measure of swings, since it has enough data points
// to mean something. The per-night streaks are what a player actually feels and
// talks about, a night being the unit the game is organized around.
func (r *PlayerRepo) GetPlayerVariance(ctx context.Context, playerID int64, from, to time.Time) (*PlayerVariance, error) {
	games, err := r.GetPlayerGameResults(ctx, playerID, from, to)
	if err != nil {
		return nil, err
	}

	sessions, err := r.GetPlayerSessionResults(ctx, playerID, from, to)
	if err != nil {
		return nil, err
	}

	gameInput := make([]StreakInput, 0, len(games))
	for _, g := range games {
		gameInput = append(gameInput, StreakInput{NetCents: g.NetCents, At: g.Created})
	}

	sessionInput := make([]StreakInput, 0, len(sessions))
	for _, s := range sessions {
		sessionInput = append(sessionInput, StreakInput{NetCents: s.NetCents, At: s.Created})
	}

	byGame, gameStreaks := spreadAndStreaks(gameInput)
	bySession, sessionStreaks := spreadAndStreaks(sessionInput)

	return &PlayerVariance{
		ByGame:         byGame,
		BySession:      bySession,
		GameStreaks:    gameStreaks,
		SessionStreaks: sessionStreaks,
	}, nil
}

// spreadAndStreaks derives both summaries from one chronological series. The
// spread needs only the amounts, which StreakInput already carries, so this keeps
// the two from being extracted separately and drifting apart.
func spreadAndStreaks(results []StreakInput) (Spread, Streaks) {
	values := make([]int, 0, len(results))
	for _, res := range results {
		values = append(values, res.NetCents)
	}

	return NewSpread(values), ComputeStreaks(results)
}
