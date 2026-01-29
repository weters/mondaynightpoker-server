package guts

import (
	"testing"
	"time"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewGame(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	assert.NoError(t, err)
	assert.NotNil(t, g)

	assert.Equal(t, 2, len(g.participants))
	assert.Equal(t, int64(1), g.idToParticipant[1].PlayerID)
	assert.Equal(t, int64(2), g.idToParticipant[2].PlayerID)
}

func TestNewGame_PlayerCount(t *testing.T) {
	// Too few players
	g, err := NewGame(logrus.StandardLogger(), []int64{1}, DefaultOptions())
	assert.Nil(t, g)
	assert.EqualError(t, err, "expected 2–10 players, got 1")

	// Too many players
	playerIDs := make([]int64, 11)
	for i := range playerIDs {
		playerIDs[i] = int64(i + 1)
	}
	g, err = NewGame(logrus.StandardLogger(), playerIDs, DefaultOptions())
	assert.Nil(t, g)
	assert.EqualError(t, err, "expected 2–10 players, got 11")

	// Valid player counts
	for count := 2; count <= 10; count++ {
		pids := make([]int64, count)
		for i := range pids {
			pids[i] = int64(i + 1)
		}
		g, err = NewGame(logrus.StandardLogger(), pids, DefaultOptions())
		assert.NoError(t, err, "should allow %d players", count)
		assert.NotNil(t, g)
	}
}

func TestNewGame_AnteDeducted(t *testing.T) {
	opts := Options{Ante: 50, MaxOwed: 1000}
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	assert.NoError(t, err)

	// Each player should have paid ante
	for _, p := range g.participants {
		assert.Equal(t, -50, p.balance)
	}

	// Pot should be total antes
	assert.Equal(t, 150, g.pot)
}

func TestGame_Deal(t *testing.T) {
	g, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())

	err := g.Deal()
	assert.NoError(t, err)

	// Each player should have 2 cards
	for _, p := range g.participants {
		assert.Len(t, p.hand, 2)
	}

	// Should be in declaration phase
	assert.Equal(t, PhaseDeclaration, g.phase)

	// All players should have pending decisions
	assert.Len(t, g.pendingDecisions, 2)
}

func TestGame_SubmitDecision(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})

	// Player 1 decides to go in
	err := g.submitDecision(1, true)
	assert.NoError(t, err)
	assert.False(t, g.pendingDecisions[1])
	assert.True(t, g.decisions[1])

	// Player 1 can't decide again
	err = g.submitDecision(1, false)
	assert.Equal(t, ErrAlreadyDecided, err)

	// Player 2 decides to go out
	err = g.submitDecision(2, false)
	assert.NoError(t, err)
	assert.False(t, g.pendingDecisions[2])
	assert.False(t, g.decisions[2])

	// All have decided - should schedule showdown
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionShowdown, g.pendingDealerAction.Action)
}

func TestGame_SubmitDecision_WrongPhase(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})
	g.phase = PhaseShowdown

	err := g.submitDecision(1, true)
	assert.Equal(t, ErrNotInDeclarationPhase, err)
}

func TestGame_Showdown_NoOneIn(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})

	// Both players go out
	_ = g.submitDecision(1, false)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()

	assert.NotNil(t, g.showdownResult)
	assert.True(t, g.showdownResult.AllFolded)
	assert.Empty(t, g.showdownResult.Winners)
	assert.Empty(t, g.showdownResult.PlayersIn)

	// Should schedule next round (re-ante)
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionNextRound, g.pendingDealerAction.Action)
}

func TestGame_Showdown_OnePersonIn(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})
	initialPot := g.pot

	// Only player 1 goes in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()

	assert.NotNil(t, g.showdownResult)
	assert.True(t, g.showdownResult.SingleWinner)
	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
	assert.Equal(t, initialPot, g.showdownResult.PotWon)

	// Winner should have received the pot
	assert.Equal(t, initialPot-25, g.idToParticipant[1].balance) // -25 ante + pot

	// Game should end
	assert.Equal(t, PhaseGameOver, g.phase)
}

func TestGame_Showdown_MultipleIn_SingleWinner(t *testing.T) {
	// Player 1 has pair of aces, player 2 has high card
	g := setupTestGame(t, []string{"14c,14d", "13c,12d"})
	initialPot := g.pot

	// Both go in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, true)

	g.calculateShowdown()

	assert.NotNil(t, g.showdownResult)
	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
	assert.Len(t, g.showdownResult.Losers, 1)
	assert.Equal(t, int64(2), g.showdownResult.Losers[0].PlayerID)

	// Winner gets pot
	assert.Equal(t, initialPot-25, g.idToParticipant[1].balance)

	// Loser pays penalty (pot amount capped at maxOwed)
	penalty := g.calculatePenalty()
	assert.Equal(t, -25-penalty, g.idToParticipant[2].balance)

	// Next pot should be the penalty
	assert.Equal(t, penalty, g.showdownResult.NextPot)
}

func TestGame_Showdown_Tie(t *testing.T) {
	// Both players have Ace-King
	g := setupTestGame(t, []string{"14c,13c", "14d,13d"})
	initialPot := g.pot

	// Both go in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, true)

	g.calculateShowdown()

	// Both should be winners
	assert.Len(t, g.showdownResult.Winners, 2)
	assert.Empty(t, g.showdownResult.Losers)

	// Pot should be split
	assert.Equal(t, initialPot/2-25, g.idToParticipant[1].balance)
	assert.Equal(t, initialPot/2-25, g.idToParticipant[2].balance)

	// No penalty, game ends
	assert.Equal(t, 0, g.showdownResult.NextPot)
	assert.Equal(t, PhaseGameOver, g.phase)
}

func TestGame_Showdown_ThreePlayersOneWinner(t *testing.T) {
	// Player 1: Pair of Aces, Player 2: Pair of Kings, Player 3: High Card
	g := setupTestGame(t, []string{"14c,14d", "13c,13d", "12c,11c"})
	initialPot := g.pot

	// All go in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, true)
	_ = g.submitDecision(3, true)

	g.calculateShowdown()

	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
	assert.Len(t, g.showdownResult.Losers, 2)

	// Winner gets pot
	assert.Equal(t, initialPot-25, g.idToParticipant[1].balance)

	// Each loser pays penalty (use showdownResult.PenaltyPaid since pot is updated after showdown)
	penalty := g.showdownResult.PenaltyPaid
	assert.Equal(t, initialPot, penalty) // Penalty should equal original pot
	assert.Equal(t, -25-penalty, g.idToParticipant[2].balance)
	assert.Equal(t, -25-penalty, g.idToParticipant[3].balance)

	// Next pot is sum of penalties
	assert.Equal(t, penalty*2, g.showdownResult.NextPot)
}

func TestGame_PenaltyCap(t *testing.T) {
	g := setupTestGame(t, []string{"14c,14d", "13c,13d"})
	g.pot = 2000 // Pot exceeds maxOwed
	g.options.MaxOwed = 1000

	penalty := g.calculatePenalty()
	assert.Equal(t, 1000, penalty, "Penalty should be capped at maxOwed")
}

func TestGame_NextRound_AllFolded(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})
	initialPot := g.pot

	// Both fold
	_ = g.submitDecision(1, false)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()

	// Manually trigger next round
	g.pendingDealerAction = nil
	err := g.nextRound()
	assert.NoError(t, err)

	// Pot should have increased by re-antes
	expectedPot := initialPot + 2*g.options.Ante
	assert.Equal(t, expectedPot, g.pot)

	// Round number should increment
	assert.Equal(t, 2, g.roundNumber)

	// Should be in declaration phase again
	assert.Equal(t, PhaseDeclaration, g.phase)
}

func TestGame_Action(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})

	// Valid decide action
	response, updateState, err := g.Action(1, &playable.PayloadIn{
		Action:         "decide",
		AdditionalData: playable.AdditionalData{"in": true},
	})
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, updateState) // Always broadcast state so players see "Decided" badge

	// Second player decides
	_, updateState, err = g.Action(2, &playable.PayloadIn{
		Action:         "decide",
		AdditionalData: playable.AdditionalData{"in": false},
	})
	assert.NoError(t, err)
	assert.True(t, updateState) // All decided

	// Unknown action
	_, _, err = g.Action(1, &playable.PayloadIn{
		Action: "unknown",
	})
	assert.Error(t, err)

	// Missing in parameter
	g2 := setupTestGame(t, []string{"14c,13c", "12d,11d"})
	_, _, err = g2.Action(1, &playable.PayloadIn{
		Action:         "decide",
		AdditionalData: playable.AdditionalData{},
	})
	assert.Error(t, err)
}

func TestGame_Action_PlayerNotFound(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})

	_, _, err := g.Action(999, &playable.PayloadIn{
		Action:         "decide",
		AdditionalData: playable.AdditionalData{"in": true},
	})
	assert.Equal(t, ErrPlayerNotFound, err)
}

func TestGame_Action_GameOver(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})
	g.phase = PhaseGameOver

	_, _, err := g.Action(1, &playable.PayloadIn{
		Action:         "decide",
		AdditionalData: playable.AdditionalData{"in": true},
	})
	assert.Equal(t, ErrGameIsOver, err)
}

func TestGame_Name(t *testing.T) {
	g := &Game{}
	assert.Equal(t, "guts", g.Name())
}

func TestGame_GetEndOfGameDetails_NotOver(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})

	details, isOver := g.GetEndOfGameDetails()
	assert.Nil(t, details)
	assert.False(t, isOver)
}

func TestGame_GetEndOfGameDetails_Over(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})
	g.done = true
	g.idToParticipant[1].balance = 100
	g.idToParticipant[2].balance = -50

	details, isOver := g.GetEndOfGameDetails()
	assert.True(t, isOver)
	assert.NotNil(t, details)
	assert.Equal(t, 100, details.BalanceAdjustments[1])
	assert.Equal(t, -50, details.BalanceAdjustments[2])
}

func TestGame_Tick(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})

	// No pending action
	updated, err := g.Tick()
	assert.NoError(t, err)
	assert.False(t, updated)

	// With pending action that's not ready (far in the future)
	g.pendingDealerAction = &pendingDealerAction{
		Action:       dealerActionShowdown,
		ExecuteAfter: time.Now().Add(time.Hour),
	}
	updated, err = g.Tick()
	assert.NoError(t, err)
	assert.False(t, updated)
}

func TestGame_Tick_Done(t *testing.T) {
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})
	g.done = true

	updated, err := g.Tick()
	assert.NoError(t, err)
	assert.False(t, updated)
}

func TestGame_Tick_ShowdownSchedulesNextRound(t *testing.T) {
	// This test verifies that Tick() clears pendingDealerAction BEFORE
	// executing the action, so any new action scheduled during execution
	// is preserved.

	// Player 1 has pair of aces, player 2 has high card
	g := setupTestGame(t, []string{"14c,14d", "13c,12d"})

	// Both go in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, true)

	// All decided - should have scheduled showdown
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionShowdown, g.pendingDealerAction.Action)

	// Set the action to execute immediately
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)

	// Execute Tick - this will run calculateShowdown()
	updated, err := g.Tick()
	assert.NoError(t, err)
	assert.True(t, updated)

	// Verify fix: calculateShowdown() should have scheduled a next round
	// action (since there was a loser who paid penalty). Before the fix,
	// this would be nil because Tick() cleared the new action.
	assert.NotNil(t, g.pendingDealerAction, "pendingDealerAction should be set for next round")
	assert.Equal(t, dealerActionNextRound, g.pendingDealerAction.Action, "should have scheduled next round")
}

func TestGame_Tick_AllFoldedSchedulesNextRound(t *testing.T) {
	// Similar test but for the all-folded case
	g := setupTestGame(t, []string{"14c,13c", "12d,11d"})

	// Both fold
	_ = g.submitDecision(1, false)
	_ = g.submitDecision(2, false)

	// Should have scheduled showdown
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionShowdown, g.pendingDealerAction.Action)

	// Set the action to execute immediately
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)

	// Execute Tick
	updated, err := g.Tick()
	assert.NoError(t, err)
	assert.True(t, updated)

	// Should have scheduled next round (re-ante)
	assert.NotNil(t, g.pendingDealerAction, "pendingDealerAction should be set for next round")
	assert.Equal(t, dealerActionNextRound, g.pendingDealerAction.Action, "should have scheduled next round for re-ante")
}

func TestNameFromOptions(t *testing.T) {
	assert.Equal(t, "2-Card Guts", NameFromOptions(DefaultOptions()))

	opts3Card := Options{Ante: 25, MaxOwed: 1000, CardCount: 3}
	assert.Equal(t, "3-Card Guts", NameFromOptions(opts3Card))

	// Default to 2-card if CardCount is not set
	optsNoCardCount := Options{Ante: 25, MaxOwed: 1000}
	assert.Equal(t, "2-Card Guts", NameFromOptions(optsNoCardCount))
}

func TestGame_Deal_3Card(t *testing.T) {
	opts := Options{Ante: 25, MaxOwed: 1000, CardCount: 3}
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	err = g.Deal()
	assert.NoError(t, err)

	// Each player should have 3 cards
	for _, p := range g.participants {
		assert.Len(t, p.hand, 3)
	}
}

func TestGame_Deal_InvalidCardCount(t *testing.T) {
	// CardCount of 1 should default to 2
	opts := Options{Ante: 25, MaxOwed: 1000, CardCount: 1}
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	err = g.Deal()
	assert.NoError(t, err)

	for _, p := range g.participants {
		assert.Len(t, p.hand, 2)
	}

	// CardCount of 4 should default to 2
	opts = Options{Ante: 25, MaxOwed: 1000, CardCount: 4}
	g, err = NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.NoError(t, err)

	err = g.Deal()
	assert.NoError(t, err)

	for _, p := range g.participants {
		assert.Len(t, p.hand, 2)
	}
}

func TestGame_Showdown_3Card(t *testing.T) {
	// Test 3-card guts showdown with straights and flushes
	// Player 1 has a straight (Q-K-A), Player 2 has a flush (all clubs)
	g := setupTestGame3Card(t, []string{"12c,13d,14h", "14c,10c,5c"})

	// Both go in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, true)

	g.calculateShowdown()

	// Player 1 should win (straight beats flush in 3-card poker)
	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
	assert.Len(t, g.showdownResult.Losers, 1)
	assert.Equal(t, int64(2), g.showdownResult.Losers[0].PlayerID)
}

func TestGame_Showdown_3Card_ThreeOfAKind(t *testing.T) {
	// Player 1 has three of a kind, Player 2 has a straight
	g := setupTestGame3Card(t, []string{"7c,7d,7h", "12c,13d,14h"})

	// Both go in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, true)

	g.calculateShowdown()

	// Player 1 should win (three of a kind beats straight)
	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
}

// Helper function to set up a 3-card test game
func setupTestGame3Card(t *testing.T, hands []string) *Game {
	t.Helper()

	playerIDs := make([]int64, len(hands))
	for i := range hands {
		playerIDs[i] = int64(i + 1)
	}

	opts := Options{Ante: 25, MaxOwed: 1000, CardCount: 3}
	g, err := NewGame(logrus.StandardLogger(), playerIDs, opts)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Set up specific hands
	for i, handStr := range hands {
		cards := deck.CardsFromString(handStr)
		g.participants[i].hand = cards
	}

	// Set up for declaration phase
	g.phase = PhaseDeclaration
	g.pendingDecisions = make(map[int64]bool)
	g.decisions = make(map[int64]bool)
	for _, p := range g.participants {
		g.pendingDecisions[p.PlayerID] = true
	}

	return g
}

// Helper function to set up a test game with specific hands
func setupTestGame(t *testing.T, hands []string) *Game {
	t.Helper()

	playerIDs := make([]int64, len(hands))
	for i := range hands {
		playerIDs[i] = int64(i + 1)
	}

	g, err := NewGame(logrus.StandardLogger(), playerIDs, DefaultOptions())
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Set up specific hands
	for i, handStr := range hands {
		cards := deck.CardsFromString(handStr)
		g.participants[i].hand = cards
	}

	// Set up for declaration phase
	g.phase = PhaseDeclaration
	g.pendingDecisions = make(map[int64]bool)
	g.decisions = make(map[int64]bool)
	for _, p := range g.participants {
		g.pendingDecisions[p.PlayerID] = true
	}

	return g
}
