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

// Helper function to set up a Bloody Guts test game with specific hands and deck
func setupBloodyGutsTestGameWithCardCount(t *testing.T, hands []string, deckCards string, cardCount int) *Game {
	t.Helper()

	playerIDs := make([]int64, len(hands))
	for i := range hands {
		playerIDs[i] = int64(i + 1)
	}

	opts := Options{Ante: 25, MaxOwed: 1000, CardCount: cardCount, BloodyGuts: true}
	g, err := NewGame(logrus.StandardLogger(), playerIDs, opts)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Set up specific hands
	for i, handStr := range hands {
		cards := deck.CardsFromString(handStr)
		g.participants[i].hand = cards
	}

	// Set up deck so the first cards drawn match what we want
	deckCardsList := deck.CardsFromString(deckCards)
	g.deck = deck.New()
	g.deck.Cards = append(deckCardsList, g.deck.Cards...)

	// Set up for declaration phase
	g.phase = PhaseDeclaration
	g.pendingDecisions = make(map[int64]bool)
	g.decisions = make(map[int64]bool)
	for _, p := range g.participants {
		g.pendingDecisions[p.PlayerID] = true
	}

	return g
}

func setupBloodyGutsTestGame(t *testing.T, hands []string, deckCards string) *Game {
	return setupBloodyGutsTestGameWithCardCount(t, hands, deckCards, 2)
}

func setupBloodyGutsTestGame3Card(t *testing.T, hands []string, deckCards string) *Game {
	return setupBloodyGutsTestGameWithCardCount(t, hands, deckCards, 3)
}

// runBloodyGutsRevealSequence runs through the entire Bloody Guts reveal sequence
// This should be called after calculateShowdown() when testing Bloody Guts with one player in
func runBloodyGutsRevealSequence(t *testing.T, g *Game) {
	t.Helper()

	// Execute all pending actions until we get past resolution
	for g.pendingDealerAction != nil {
		action := g.pendingDealerAction.Action
		g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
		_, err := g.Tick()
		assert.NoError(t, err)

		// Stop after we've resolved (next action will be next round or end game)
		if action == dealerActionResolveBloodyGuts {
			break
		}
	}
}

func TestBloodyGuts_PlayerBeatsDeck(t *testing.T) {
	// Player has Ace-King, deck has Queen-Jack
	g := setupBloodyGutsTestGame(t, []string{"14c,13c", "12d,11d"}, "12h,11h")
	initialPot := g.pot

	// Only player 1 goes in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()
	runBloodyGutsRevealSequence(t, g)

	assert.NotNil(t, g.showdownResult)
	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
	assert.False(t, g.showdownResult.DeckWon)
	assert.NotNil(t, g.showdownResult.DeckHand)
	assert.Len(t, g.showdownResult.DeckHand, 2)

	// Winner should have received the pot
	assert.Equal(t, initialPot-25, g.idToParticipant[1].balance)

	// Game should end
	assert.Equal(t, PhaseGameOver, g.phase)
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionEndGame, g.pendingDealerAction.Action)
}

func TestBloodyGuts_DeckBeatsPlayer(t *testing.T) {
	// Player has Queen-Jack, deck has Ace-King
	g := setupBloodyGutsTestGame(t, []string{"12c,11c", "10d,9d"}, "14h,13h")
	initialPot := g.pot

	// Only player 1 goes in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()
	runBloodyGutsRevealSequence(t, g)

	assert.NotNil(t, g.showdownResult)
	assert.True(t, g.showdownResult.DeckWon)
	assert.Empty(t, g.showdownResult.Winners)
	assert.Len(t, g.showdownResult.Losers, 1)
	assert.Equal(t, int64(1), g.showdownResult.Losers[0].PlayerID)
	assert.NotNil(t, g.showdownResult.DeckHand)

	// Player pays penalty
	penalty := initialPot // Penalty equals pot when below maxOwed
	assert.Equal(t, -25-penalty, g.idToParticipant[1].balance)
	assert.Equal(t, penalty, g.showdownResult.PenaltyPaid)
	// In Bloody Guts, when deck wins, pot accumulates (original pot + penalty)
	assert.Equal(t, initialPot+penalty, g.showdownResult.NextPot)
	assert.Equal(t, initialPot+penalty, g.pot)

	// Game continues to next round
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionNextRound, g.pendingDealerAction.Action)
}

func TestBloodyGuts_DeckWinsOnTie(t *testing.T) {
	// Player has Ace-King, deck has Ace-King (same strength)
	g := setupBloodyGutsTestGame(t, []string{"14c,13c", "12d,11d"}, "14h,13h")
	initialPot := g.pot

	// Only player 1 goes in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()
	runBloodyGutsRevealSequence(t, g)

	assert.NotNil(t, g.showdownResult)
	assert.True(t, g.showdownResult.DeckWon, "Deck should win on tie")
	assert.Empty(t, g.showdownResult.Winners)
	assert.Len(t, g.showdownResult.Losers, 1)

	// Player pays penalty
	penalty := initialPot
	assert.Equal(t, -25-penalty, g.idToParticipant[1].balance)

	// Game continues
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionNextRound, g.pendingDealerAction.Action)
}

func TestBloodyGuts_MultiplePlayersIn(t *testing.T) {
	// When multiple players go in, normal showdown occurs (no deck involved)
	g := setupBloodyGutsTestGame(t, []string{"14c,14d", "13c,12d"}, "10h,9h")
	initialPot := g.pot

	// Both go in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, true)

	g.calculateShowdown()

	assert.NotNil(t, g.showdownResult)
	// Player 1 wins with pair
	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
	// Deck should not be involved
	assert.Nil(t, g.showdownResult.DeckHand)
	assert.False(t, g.showdownResult.DeckWon)
	// Player 2 is a loser
	assert.Len(t, g.showdownResult.Losers, 1)
	assert.Equal(t, int64(2), g.showdownResult.Losers[0].PlayerID)
	// Winner gets pot
	assert.Equal(t, initialPot-25, g.idToParticipant[1].balance)
}

func TestBloodyGuts_NoOneIn(t *testing.T) {
	// When no one goes in, normal re-ante behavior
	g := setupBloodyGutsTestGame(t, []string{"14c,13c", "12d,11d"}, "10h,9h")

	// Both fold
	_ = g.submitDecision(1, false)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()

	assert.NotNil(t, g.showdownResult)
	assert.True(t, g.showdownResult.AllFolded)
	assert.Nil(t, g.showdownResult.DeckHand)
	assert.False(t, g.showdownResult.DeckWon)

	// Schedule next round for re-ante
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionNextRound, g.pendingDealerAction.Action)
}

func TestBloodyGuts_3Card(t *testing.T) {
	// Test 3-card Bloody Guts - player has straight, deck has high card
	g := setupBloodyGutsTestGame3Card(t, []string{"12c,13d,14h", "5c,7d,9h"}, "14s,10s,5s")
	initialPot := g.pot

	// Only player 1 goes in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()
	runBloodyGutsRevealSequence(t, g)

	assert.NotNil(t, g.showdownResult)
	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
	assert.False(t, g.showdownResult.DeckWon)
	// Verify 3 cards were drawn
	assert.Len(t, g.showdownResult.DeckHand, 3)
	// Winner gets pot
	assert.Equal(t, initialPot-25, g.idToParticipant[1].balance)
}

func TestBloodyGuts_PenaltyCapped(t *testing.T) {
	// Test that penalty is capped at MaxOwed when losing to deck
	g := setupBloodyGutsTestGame(t, []string{"10c,9c", "8d,7d"}, "14h,13h")
	g.pot = 2000            // Pot exceeds maxOwed
	g.options.MaxOwed = 500 // Set low maxOwed

	// Only player 1 goes in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()
	runBloodyGutsRevealSequence(t, g)

	assert.True(t, g.showdownResult.DeckWon)
	assert.Equal(t, 500, g.showdownResult.PenaltyPaid, "Penalty should be capped at maxOwed")
	// In Bloody Guts, pot accumulates: original pot (2000) + capped penalty (500)
	assert.Equal(t, 2500, g.showdownResult.NextPot)
	// Player balance: -25 ante - 500 penalty = -525
	assert.Equal(t, -25-500, g.idToParticipant[1].balance)
}

func TestBloodyGuts_DeckHandCleared(t *testing.T) {
	// Test that deckHand is cleared between rounds
	g := setupBloodyGutsTestGame(t, []string{"10c,9c", "8d,7d"}, "14h,13h")

	// Only player 1 goes in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	g.calculateShowdown()
	runBloodyGutsRevealSequence(t, g)

	// Deck won, deckHand should be set
	assert.NotNil(t, g.deckHand)
	assert.Len(t, g.deckHand, 2)

	// Simulate next round
	g.pendingDealerAction = nil
	err := g.nextRound()
	assert.NoError(t, err)

	// deckHand should be cleared
	assert.Nil(t, g.deckHand)
}

func TestNameFromOptions_BloodyGuts(t *testing.T) {
	// 2-card Bloody Guts
	opts2Card := Options{Ante: 25, MaxOwed: 1000, CardCount: 2, BloodyGuts: true}
	assert.Equal(t, "Bloody 2-Card Guts", NameFromOptions(opts2Card))

	// 3-card Bloody Guts
	opts3Card := Options{Ante: 25, MaxOwed: 1000, CardCount: 3, BloodyGuts: true}
	assert.Equal(t, "Bloody 3-Card Guts", NameFromOptions(opts3Card))

	// Non-bloody versions still work
	assert.Equal(t, "2-Card Guts", NameFromOptions(DefaultOptions()))
	opts3CardNonBloody := Options{Ante: 25, MaxOwed: 1000, CardCount: 3, BloodyGuts: false}
	assert.Equal(t, "3-Card Guts", NameFromOptions(opts3CardNonBloody))
}

func TestBloodyGuts_RevealSequence(t *testing.T) {
	// Player has Ace-King, deck has Queen-Jack (player will win)
	g := setupBloodyGutsTestGame(t, []string{"14c,13c", "12d,11d"}, "12h,11h")

	// Only player 1 goes in
	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	// After decisions, should schedule showdown
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionShowdown, g.pendingDealerAction.Action)

	// Execute showdown
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	updated, err := g.Tick()
	assert.NoError(t, err)
	assert.True(t, updated)

	// Should have drawn deck cards but not revealed any yet
	assert.NotNil(t, g.deckHand)
	assert.Len(t, g.deckHand, 2)
	assert.Equal(t, 0, g.deckCardsRevealed)
	assert.Equal(t, int64(1), g.bloodyGutsPlayer)

	// Should schedule first card reveal
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionRevealDeckCard, g.pendingDealerAction.Action)

	// First reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	updated, err = g.Tick()
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, 1, g.deckCardsRevealed)

	// Should schedule second card reveal
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionRevealDeckCard, g.pendingDealerAction.Action)

	// Second reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	updated, err = g.Tick()
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, 2, g.deckCardsRevealed)

	// Should schedule resolution
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionResolveBloodyGuts, g.pendingDealerAction.Action)

	// Winner should NOT be determined yet
	assert.Nil(t, g.showdownResult.Winners)
	assert.False(t, g.showdownResult.DeckWon)
}

func TestBloodyGuts_RevealTiming(t *testing.T) {
	g := setupBloodyGutsTestGame(t, []string{"14c,13c", "12d,11d"}, "12h,11h")

	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	// Execute showdown
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	// First reveal should be scheduled 2 seconds in the future
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionRevealDeckCard, g.pendingDealerAction.Action)
	assert.True(t, g.pendingDealerAction.ExecuteAfter.After(time.Now().Add(time.Second)))
	assert.True(t, g.pendingDealerAction.ExecuteAfter.Before(time.Now().Add(time.Second*3)))

	// Execute first reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	// Second reveal should also be scheduled 2 seconds in the future
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionRevealDeckCard, g.pendingDealerAction.Action)
	assert.True(t, g.pendingDealerAction.ExecuteAfter.After(time.Now().Add(time.Second)))
	assert.True(t, g.pendingDealerAction.ExecuteAfter.Before(time.Now().Add(time.Second*3)))

	// Execute second reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	// Resolution should be scheduled 1 second in the future
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionResolveBloodyGuts, g.pendingDealerAction.Action)
	assert.True(t, g.pendingDealerAction.ExecuteAfter.After(time.Now()))
	assert.True(t, g.pendingDealerAction.ExecuteAfter.Before(time.Now().Add(time.Second*2)))
}

func TestBloodyGuts_WinnerAfterLastCard(t *testing.T) {
	// Player has Ace-King, deck has Queen-Jack (player wins)
	g := setupBloodyGutsTestGame(t, []string{"14c,13c", "12d,11d"}, "12h,11h")
	initialPot := g.pot

	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	// Execute showdown
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	// Reveal cards
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick() // First card
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick() // Second card

	// Winner should NOT be determined yet
	assert.Empty(t, g.showdownResult.Winners)
	assert.Equal(t, 0, g.showdownResult.PotWon)

	// Execute resolution
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	// NOW winner should be determined
	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
	assert.False(t, g.showdownResult.DeckWon)
	assert.Equal(t, initialPot, g.showdownResult.PotWon)
	assert.Equal(t, initialPot-25, g.idToParticipant[1].balance)
	assert.Equal(t, PhaseGameOver, g.phase)
}

func TestBloodyGuts_NextRoundAfterResolve(t *testing.T) {
	// Player has Queen-Jack, deck has Ace-King (deck wins)
	g := setupBloodyGutsTestGame(t, []string{"12c,11c", "10d,9d"}, "14h,13h")

	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	// Execute through reveal sequence
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick() // Showdown
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick() // First reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick() // Second reveal

	// Before resolution, next round should NOT be scheduled
	assert.Equal(t, dealerActionResolveBloodyGuts, g.pendingDealerAction.Action)

	// Execute resolution
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	// Deck won, next round should be scheduled
	assert.True(t, g.showdownResult.DeckWon)
	assert.NotNil(t, g.pendingDealerAction)
	assert.Equal(t, dealerActionNextRound, g.pendingDealerAction.Action)
}

func TestBloodyGuts_3Card_RevealSequence(t *testing.T) {
	// 3-card game: player has straight, deck has high card
	g := setupBloodyGutsTestGame3Card(t, []string{"12c,13d,14h", "5c,7d,9h"}, "14s,10s,5s")

	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	// Execute showdown
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	assert.Len(t, g.deckHand, 3)
	assert.Equal(t, 0, g.deckCardsRevealed)

	// First reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()
	assert.Equal(t, 1, g.deckCardsRevealed)
	assert.Equal(t, dealerActionRevealDeckCard, g.pendingDealerAction.Action)

	// Second reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()
	assert.Equal(t, 2, g.deckCardsRevealed)
	assert.Equal(t, dealerActionRevealDeckCard, g.pendingDealerAction.Action)

	// Third reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()
	assert.Equal(t, 3, g.deckCardsRevealed)
	assert.Equal(t, dealerActionResolveBloodyGuts, g.pendingDealerAction.Action)

	// Execute resolution
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	// Player wins with straight
	assert.Len(t, g.showdownResult.Winners, 1)
	assert.Equal(t, int64(1), g.showdownResult.Winners[0].PlayerID)
	assert.False(t, g.showdownResult.DeckWon)
}

func TestBloodyGuts_StateOnlyShowsRevealed(t *testing.T) {
	g := setupBloodyGutsTestGame(t, []string{"14c,13c", "12d,11d"}, "12h,11h")

	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	// Execute showdown
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	// Before any reveals - state should show no cards
	state := g.getGameState()
	assert.Nil(t, state.DeckHand)
	assert.Equal(t, 0, state.DeckCardsRevealed)
	assert.Equal(t, 2, state.DeckCardsTotal)

	// First reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	state = g.getGameState()
	assert.Len(t, state.DeckHand, 1)
	assert.Equal(t, 1, state.DeckCardsRevealed)
	assert.Equal(t, 2, state.DeckCardsTotal)

	// Second reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	state = g.getGameState()
	assert.Len(t, state.DeckHand, 2)
	assert.Equal(t, 2, state.DeckCardsRevealed)
	assert.Equal(t, 2, state.DeckCardsTotal)
}

func TestBloodyGuts_RevealFieldsClearedOnNextRound(t *testing.T) {
	// Deck wins so game continues
	g := setupBloodyGutsTestGame(t, []string{"12c,11c", "10d,9d"}, "14h,13h")

	_ = g.submitDecision(1, true)
	_ = g.submitDecision(2, false)

	// Execute through reveal and resolution
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick() // Showdown
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick() // First reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick() // Second reveal
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick() // Resolution

	// Verify fields are set
	assert.Equal(t, 2, g.deckCardsRevealed)
	assert.Equal(t, int64(1), g.bloodyGutsPlayer)

	// Execute next round
	g.pendingDealerAction.ExecuteAfter = time.Now().Add(-time.Second)
	_, _ = g.Tick()

	// Fields should be cleared
	assert.Equal(t, 0, g.deckCardsRevealed)
	assert.Equal(t, int64(0), g.bloodyGutsPlayer)
	assert.Nil(t, g.deckHand)
}
