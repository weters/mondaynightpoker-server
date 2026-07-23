package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlayerTransactions(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true // to rapidly create tables
	tbl1, err := testRepos.Tables.CreateTable(cbg, p, "Ledger Table 1")
	require.NoError(t, err)
	tbl2, err := testRepos.Tables.CreateTable(cbg, p, "Ledger Table 2")
	require.NoError(t, err)

	// two game-driven adjustments at tbl1
	g1, _ := testRepos.Games.CreateGame(cbg, tbl1, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, g1, nil, map[int64]int{p.ID: 100}))
	g2, _ := testRepos.Games.CreateGame(cbg, tbl1, "Texas Hold'em (${25}/${50})")
	require.NoError(t, testRepos.Games.EndGame(cbg, g2, nil, map[int64]int{p.ID: -30}))

	// one at tbl2
	g3, _ := testRepos.Games.CreateGame(cbg, tbl2, "guts")
	require.NoError(t, testRepos.Games.EndGame(cbg, g3, nil, map[int64]int{p.ID: 50}))

	// all transactions, newest first
	all, err := testRepos.Players.GetPlayerTransactions(cbg, p.ID, nil, 0, 100)
	a.NoError(err)
	a.Equal(3, len(all))

	// newest first: g3, g2, g1 (created ASC then id ASC, so DESC lists g3 first)
	a.Equal(g3.ID, *all[0].GameID)
	a.Equal(50, all[0].Adjustment)
	a.Equal(tbl2.UUID, all[0].TableUUID)
	a.Equal("Ledger Table 2", all[0].TableName)
	require.NotNil(t, all[0].GameType)
	a.Equal("guts", *all[0].GameType)
	a.Equal("game ended", all[0].Reason)

	// previous/current balances track the running balance
	a.Equal(g2.ID, *all[1].GameID)
	a.Equal(-30, all[1].Adjustment)
	a.Equal(100, all[1].PreviousBalance)
	a.Equal(70, all[1].CurrentBalance)

	count, err := testRepos.Players.GetPlayerTransactionsCount(cbg, p.ID, nil)
	a.NoError(err)
	a.Equal(int64(3), count)

	// filtered to a single table
	filtered, err := testRepos.Players.GetPlayerTransactions(cbg, p.ID, &tbl1.UUID, 0, 100)
	a.NoError(err)
	a.Equal(2, len(filtered))
	for _, tx := range filtered {
		a.Equal(tbl1.UUID, tx.TableUUID)
	}

	fCount, err := testRepos.Players.GetPlayerTransactionsCount(cbg, p.ID, &tbl1.UUID)
	a.NoError(err)
	a.Equal(int64(2), fCount)

	// pagination
	page, err := testRepos.Players.GetPlayerTransactions(cbg, p.ID, nil, 1, 1)
	a.NoError(err)
	a.Equal(1, len(page))
	a.Equal(g2.ID, *page[0].GameID)
}

// TestGetPlayerTransactions_ExcludesDeletedTables pins the invariant that ledger
// rows at a soft-deleted table are invisible, in both the list and the count.
func TestGetPlayerTransactions_ExcludesDeletedTables(t *testing.T) {
	a := assert.New(t)

	p := player()
	p.IsSiteAdmin = true
	live, _ := testRepos.Tables.CreateTable(cbg, p, "Ledger Live")
	gone, _ := testRepos.Tables.CreateTable(cbg, p, "Ledger Deleted")

	lg, _ := testRepos.Games.CreateGame(cbg, live, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, lg, nil, map[int64]int{p.ID: 100}))
	gg, _ := testRepos.Games.CreateGame(cbg, gone, "bourre")
	require.NoError(t, testRepos.Games.EndGame(cbg, gg, nil, map[int64]int{p.ID: 700}))

	// both visible while live
	all, err := testRepos.Players.GetPlayerTransactions(cbg, p.ID, nil, 0, 100)
	a.NoError(err)
	a.Equal(2, len(all))

	gone.Deleted = true
	require.NoError(t, testRepos.Tables.Save(cbg, gone))

	// the deleted table's row is gone from list and count
	all, err = testRepos.Players.GetPlayerTransactions(cbg, p.ID, nil, 0, 100)
	a.NoError(err)
	a.Equal(1, len(all))
	a.Equal(live.UUID, all[0].TableUUID)

	count, err := testRepos.Players.GetPlayerTransactionsCount(cbg, p.ID, nil)
	a.NoError(err)
	a.Equal(int64(1), count)
}
