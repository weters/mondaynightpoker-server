package mcpserver

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mondaynightpoker-server/pkg/model"
)

// holdEmLog is a persisted Texas Hold'em log in the shape the game writes: the
// action and variant enums are objects because their MarshalJSON emits
// {"id","name"}. Building it as a literal keeps these tests independent of the
// game packages while still exercising the real on-disk encoding.
func holdEmLog(winnerID, loserID int64) map[string]interface{} {
	return map[string]interface{}{
		"variant":    map[string]interface{}{"id": "standard", "name": "Standard"},
		"ante":       0,
		"smallBlind": 25,
		"bigBlind":   50,
		"pot":        300,
		"community": []interface{}{
			map[string]interface{}{"rank": 14, "suit": "spades"},
			map[string]interface{}{"rank": 7, "suit": "diamonds"},
			map[string]interface{}{"rank": 2, "suit": "clubs"},
		},
		"seats": []interface{}{
			map[string]interface{}{"playerId": winnerID, "holeCards": []interface{}{map[string]interface{}{"rank": 13, "suit": "spades"}}},
			map[string]interface{}{"playerId": loserID, "holeCards": []interface{}{map[string]interface{}{"rank": 3, "suit": "clubs"}}},
		},
		"actions": []interface{}{
			map[string]interface{}{"street": "preflop", "playerId": winnerID, "action": map[string]interface{}{"id": "raise", "name": "Raise"}, "amount": 150},
			map[string]interface{}{"street": "preflop", "playerId": loserID, "action": map[string]interface{}{"id": "call", "name": "Call"}, "amount": 150},
			map[string]interface{}{"street": "flop", "playerId": winnerID, "action": map[string]interface{}{"id": "bet", "name": "Bet"}, "amount": 100},
			map[string]interface{}{"street": "flop", "playerId": loserID, "action": map[string]interface{}{"id": "fold", "name": "Fold"}},
		},
		"participants": []interface{}{
			map[string]interface{}{"playerId": winnerID, "winnings": 300},
			map[string]interface{}{"playerId": loserID, "folded": true},
		},
	}
}

// endHoldEmGame creates and ends a Texas Hold'em game at the table, storing a real
// log. The stored game_type is a display name, which is what the column actually
// holds and what the parser dispatch has to resolve.
func endHoldEmGame(t *testing.T, tbl *model.Table, winnerID, loserID int64, adjustments map[int64]int) *model.Game {
	t.Helper()

	game, err := testRepos.Games.CreateGame(cbg, tbl, "Texas Hold'em")
	require.NoError(t, err)
	require.NoError(t, testRepos.Games.EndGame(cbg, game, holdEmLog(winnerID, loserID), adjustments))

	return game
}

func TestGetHandHistory(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	other := createPlayer(t)

	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Hand History Table")
	a.NoError(err)
	_, err = testRepos.Tables.Join(cbg, other, tbl)
	a.NoError(err)

	game := endHoldEmGame(t, tbl, admin.ID, other.ID, map[int64]int{admin.ID: 150, other.ID: -150})

	// The table uuid is the capability, so any authenticated player holding it may
	// read the hand — same rule as get_game.
	outsider := createPlayer(t)
	out, err := s.getHandHistory(cbg, nil, playerCaller(outsider.ID), getHandHistoryInput{UUID: tbl.UUID, ID: game.ID})
	require.NoError(t, err)

	hand := out.Hand
	a.Equal(game.ID, hand.GameID)
	a.Equal("texas-hold-em", hand.GameType, "the display name resolves to a factory identifier")
	a.Equal("Texas Hold'em", hand.StoredGameType)
	a.Equal("standard", hand.Variant)
	a.Equal(300, hand.PotCents)
	require.Len(t, hand.Board, 3)
	a.Equal("A♠", hand.Board[0].Display)
	a.Equal("spades", hand.Board[0].Suit)
	a.Equal(14, hand.Board[0].Rank)
	a.Len(hand.Actions, 4)

	// Display names and net results come from the ledger, not the log.
	require.Len(t, hand.Participants, 2)
	byID := map[int64]HandParticipantDTO{}
	for _, p := range hand.Participants {
		byID[p.PlayerID] = p
	}

	winner := byID[admin.ID]
	a.True(winner.VoluntarilyPlayed)
	a.True(winner.Won)
	a.False(winner.WentToShowdown, "the opponent folded, so nothing was shown down")
	a.Equal(250, winner.AmountWageredCents)
	require.NotNil(t, winner.NetCents)
	a.Equal(150, *winner.NetCents)
	require.NotNil(t, winner.NetDisplay)
	a.Equal("$1.50", *winner.NetDisplay)
	a.NotEmpty(winner.DisplayName)

	loser := byID[other.ID]
	a.True(loser.Folded)
	a.True(loser.VoluntarilyPlayed, "called before folding later")
	require.NotNil(t, loser.NetCents)
	a.Equal(-150, *loser.NetCents)

	// Actions are normalized and ordered.
	a.Equal("raise", hand.Actions[0].Kind)
	a.Equal("$1.50", hand.Actions[0].AmountDisplay)
	a.Equal("fold", hand.Actions[3].Kind)
	a.Equal(3, hand.Actions[3].Sequence)
}

// TestGetHandHistory_WrongTableIsNotFound is the enumeration guard: games.id is
// sequential, so the table uuid has to be the capability here exactly as it is for
// get_game.
func TestGetHandHistory_WrongTableIsNotFound(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)

	owner, err := testRepos.Tables.CreateTable(cbg, admin, "Owner Table")
	a.NoError(err)
	game := endHoldEmGame(t, owner, admin.ID, admin.ID, map[int64]int{admin.ID: 100})

	other, err := testRepos.Tables.CreateTable(cbg, admin, "Other Table")
	a.NoError(err)

	_, err = s.getHandHistory(cbg, nil, adminCaller(admin.ID), getHandHistoryInput{UUID: other.UUID, ID: game.ID})
	a.Error(err)
	a.Contains(err.Error(), "game not found")

	// An unknown table uuid reports the game missing, never the table: the response
	// must not confirm the game exists.
	_, err = s.getHandHistory(cbg, nil, adminCaller(admin.ID), getHandHistoryInput{UUID: uuid.New().String(), ID: game.ID})
	a.Error(err)
	a.Contains(err.Error(), "game not found")

	_, err = s.getHandHistory(cbg, nil, adminCaller(admin.ID), getHandHistoryInput{UUID: owner.UUID, ID: -999})
	a.Error(err)
	a.Contains(err.Error(), "game not found")
}

func TestGetHandHistory_DeletedTableIsNotFound(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Doomed Table")
	a.NoError(err)
	game := endHoldEmGame(t, tbl, admin.ID, admin.ID, map[int64]int{admin.ID: 100})

	tbl.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, tbl))

	_, err = s.getHandHistory(cbg, nil, adminCaller(admin.ID), getHandHistoryInput{UUID: tbl.UUID, ID: game.ID})
	a.Error(err)
	a.Contains(err.Error(), "game not found")
}

// TestGetHandHistory_UnfinishedGame covers a game row created but never ended: it
// has no log, so there is no hand to return.
func TestGetHandHistory_UnfinishedGame(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Unfinished Table")
	a.NoError(err)

	game, err := testRepos.Games.CreateGame(cbg, tbl, "Texas Hold'em")
	a.NoError(err)

	_, err = s.getHandHistory(cbg, nil, adminCaller(admin.ID), getHandHistoryInput{UUID: tbl.UUID, ID: game.ID})
	a.Error(err)
	a.Contains(err.Error(), "hand history not found")
}

func TestGetPlayerTendencies(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	other := createPlayer(t)

	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Tendencies Table")
	a.NoError(err)
	_, err = testRepos.Tables.Join(cbg, other, tbl)
	a.NoError(err)

	// Two identical hands: admin raises and bets, other calls then folds.
	endHoldEmGame(t, tbl, admin.ID, other.ID, map[int64]int{admin.ID: 150, other.ID: -150})
	endHoldEmGame(t, tbl, admin.ID, other.ID, map[int64]int{admin.ID: 150, other.ID: -150})

	out, err := s.getPlayerTendencies(cbg, nil, adminCaller(admin.ID), getPlayerTendenciesInput{ID: admin.ID})
	require.NoError(t, err)

	a.Equal(admin.ID, out.Player.ID)
	a.Equal(2, out.GamesAnalyzed)
	a.Equal(int64(2), out.GamesInRange)
	a.False(out.Truncated)
	a.Zero(out.GamesSkipped)

	tend := out.Tendencies
	a.Equal(2, tend.HandsAnalyzed)
	a.Equal(2, tend.HandsVoluntarilyPlayed)
	a.Equal(2, tend.HandsWon)
	a.Zero(tend.HandsFolded)
	a.Equal(2, tend.Raises)
	a.Equal(2, tend.Bets)
	a.Zero(tend.Calls)
	a.Equal(500, tend.AmountWageredCents)

	require.NotNil(t, tend.VoluntaryPlayRate)
	a.Equal(1.0, tend.VoluntaryPlayRate.Value)
	a.Equal("100.0%", tend.VoluntaryPlayRate.Display)

	// Never called, so the aggression factor is undefined rather than infinite.
	a.Nil(tend.AggressionFactor, "dividing by zero calls would report infinite aggression")

	// Never reached a showdown, so that rate is absent rather than a misleading 0%.
	a.Zero(tend.HandsToShowdown)
	a.Nil(tend.ShowdownWinRate)

	// The per-game-type breakdown is keyed by factory identifier.
	require.Contains(t, out.ByGameType, "texas-hold-em")
	a.Equal(2, out.ByGameType["texas-hold-em"].HandsAnalyzed)

	// The other player called and folded, which is a different profile from the
	// same hands.
	out, err = s.getPlayerTendencies(cbg, nil, adminCaller(admin.ID), getPlayerTendenciesInput{ID: other.ID})
	require.NoError(t, err)

	tend = out.Tendencies
	a.Equal(2, tend.HandsAnalyzed)
	a.Equal(2, tend.HandsFolded)
	a.Equal(2, tend.Calls)
	a.Zero(tend.HandsWon)

	require.NotNil(t, tend.AggressionFactor)
	a.Zero(*tend.AggressionFactor, "called twice, never bet or raised")

	require.NotNil(t, tend.FoldRate)
	a.Equal("100.0%", tend.FoldRate.Display)
}

func TestGetPlayerTendencies_GameTypeFilter(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Filter Table")
	a.NoError(err)

	endHoldEmGame(t, tbl, admin.ID, admin.ID, map[int64]int{admin.ID: 150})

	out, err := s.getPlayerTendencies(cbg, nil, adminCaller(admin.ID),
		getPlayerTendenciesInput{ID: admin.ID, GameType: ptrStr("texas-hold-em")})
	require.NoError(t, err)
	a.Equal(1, out.Tendencies.HandsAnalyzed)

	// A game type the player has not played yields an empty profile, not an error.
	out, err = s.getPlayerTendencies(cbg, nil, adminCaller(admin.ID),
		getPlayerTendenciesInput{ID: admin.ID, GameType: ptrStr("bourre")})
	require.NoError(t, err)
	a.Zero(out.Tendencies.HandsAnalyzed)

	// An unknown game type is an error rather than an empty profile, which would
	// read as "this player has never played it".
	_, err = s.getPlayerTendencies(cbg, nil, adminCaller(admin.ID),
		getPlayerTendenciesInput{ID: admin.ID, GameType: ptrStr("go-fish")})
	a.Error(err)
	a.Contains(err.Error(), "no factory with name: go-fish")
}

// TestGetPlayerTendencies_SkipsUnparsableLogs pins that one unreadable payload does
// not fail the call or vanish silently.
func TestGetPlayerTendencies_SkipsUnparsableLogs(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Mixed Logs Table")
	a.NoError(err)

	endHoldEmGame(t, tbl, admin.ID, admin.ID, map[int64]int{admin.ID: 150})

	// A log whose shape the parser cannot read.
	bad, err := testRepos.Games.CreateGame(cbg, tbl, "Texas Hold'em")
	a.NoError(err)
	a.NoError(testRepos.Games.EndGame(cbg, bad, map[string]interface{}{"seats": "not-an-array"}, map[int64]int{admin.ID: 10}))

	out, err := s.getPlayerTendencies(cbg, nil, adminCaller(admin.ID), getPlayerTendenciesInput{ID: admin.ID})
	require.NoError(t, err)

	a.Equal(1, out.GamesAnalyzed)
	a.Equal(1, out.GamesSkipped, "the unreadable log is reported, not hidden")
	a.Equal(int64(2), out.GamesInRange)
}

func TestGetPlayerTendencies_SelfScoped(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	p1 := createPlayer(t)
	p2 := createPlayer(t)

	// The policy wrapper enforces scoping, so drive the registered handler.
	wrapped := wrapHandler(accessSelfScoped, s.getPlayerTendencies)

	_, _, err := wrapped(ctxForPlayer(p1.ID), nil, getPlayerTendenciesInput{ID: p2.ID})
	a.Error(err)
	a.Contains(err.Error(), "permission denied")

	_, _, err = wrapped(ctxForPlayer(p1.ID), nil, getPlayerTendenciesInput{ID: p1.ID})
	a.NoError(err)

	// An admin may target anyone.
	_, _, err = wrapped(ctxForAdmin(p1.ID), nil, getPlayerTendenciesInput{ID: p2.ID})
	a.NoError(err)
}

func TestGetPlayerVariance(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)

	night1, err := testRepos.Tables.CreateTable(cbg, admin, "Variance Night One")
	a.NoError(err)
	night2, err := testRepos.Tables.CreateTable(cbg, admin, "Variance Night Two")
	a.NoError(err)

	// Night one: +100, +50. Night two: -30.
	endHoldEmGame(t, night1, admin.ID, admin.ID, map[int64]int{admin.ID: 100})
	endHoldEmGame(t, night1, admin.ID, admin.ID, map[int64]int{admin.ID: 50})
	endHoldEmGame(t, night2, admin.ID, admin.ID, map[int64]int{admin.ID: -30})

	out, err := s.getPlayerVariance(cbg, nil, adminCaller(admin.ID), getPlayerVarianceInput{ID: admin.ID})
	require.NoError(t, err)

	a.Equal(admin.ID, out.Player.ID)

	byGame := out.Variance.ByGame
	a.Equal(3, byGame.Count)
	a.Equal(120, byGame.TotalCents)
	a.Equal("$1.20", byGame.TotalDisplay)
	a.Equal(100, byGame.BestCents)
	a.Equal(-30, byGame.WorstCents)
	a.Greater(byGame.StdDevCents, 0.0)
	a.NotEmpty(byGame.StdDevDisplay)

	// A night aggregates its games, so two nights rather than three results.
	bySession := out.Variance.BySession
	a.Equal(2, bySession.Count)
	a.Equal(150, bySession.BestCents)
	a.Equal(-30, bySession.WorstCents)

	require.NotNil(t, out.Variance.SessionStreaks.LongestWinning)
	a.Equal(1, out.Variance.SessionStreaks.LongestWinning.Length)
	a.Equal("winning", out.Variance.SessionStreaks.LongestWinning.Outcome)

	require.NotNil(t, out.Variance.SessionStreaks.Current)
	a.Equal("losing", out.Variance.SessionStreaks.Current.Outcome, "the most recent night lost")

	require.NotNil(t, out.Variance.GameStreaks.LongestWinning)
	a.Equal(2, out.Variance.GameStreaks.LongestWinning.Length)
	a.Equal(150, out.Variance.GameStreaks.LongestWinning.NetCents)
}

// TestGetPlayerVariance_NoResults pins that a player with nothing to measure gets
// zeroes and absent streaks rather than an error.
func TestGetPlayerVariance_NoResults(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	p := createPlayer(t)

	out, err := s.getPlayerVariance(cbg, nil, playerCaller(p.ID), getPlayerVarianceInput{ID: p.ID})
	require.NoError(t, err)

	a.Zero(out.Variance.ByGame.Count)
	a.Zero(out.Variance.BySession.Count)
	a.Nil(out.Variance.SessionStreaks.Current)
	a.Nil(out.Variance.SessionStreaks.LongestWinning)
	a.Nil(out.Variance.SessionStreaks.LongestLosing)
}

func TestGetPlayerVariance_SelfScoped(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	p1 := createPlayer(t)
	p2 := createPlayer(t)

	wrapped := wrapHandler(accessSelfScoped, s.getPlayerVariance)

	_, _, err := wrapped(ctxForPlayer(p1.ID), nil, getPlayerVarianceInput{ID: p2.ID})
	a.Error(err)
	a.Contains(err.Error(), "permission denied")

	_, _, err = wrapped(ctxForPlayer(p1.ID), nil, getPlayerVarianceInput{ID: p1.ID})
	a.NoError(err)

	_, _, err = wrapped(ctxForAdmin(p1.ID), nil, getPlayerVarianceInput{ID: p2.ID})
	a.NoError(err)
}

// TestGetPlayerVariance_ExcludesDeletedTables pins that a soft-deleted table is
// invisible here as it is across the rest of the read surface.
func TestGetPlayerVariance_ExcludesDeletedTables(t *testing.T) {
	a := assert.New(t)
	s := newServer()

	admin := createSiteAdmin(t)
	tbl, err := testRepos.Tables.CreateTable(cbg, admin, "Deleted Variance Table")
	a.NoError(err)

	endHoldEmGame(t, tbl, admin.ID, admin.ID, map[int64]int{admin.ID: 500})

	tbl.Deleted = true
	a.NoError(testRepos.Tables.Save(cbg, tbl))

	out, err := s.getPlayerVariance(cbg, nil, adminCaller(admin.ID), getPlayerVarianceInput{ID: admin.ID})
	require.NoError(t, err)
	a.Zero(out.Variance.ByGame.Count)

	tend, err := s.getPlayerTendencies(cbg, nil, adminCaller(admin.ID), getPlayerTendenciesInput{ID: admin.ID})
	require.NoError(t, err)
	a.Zero(tend.Tendencies.HandsAnalyzed)
}
