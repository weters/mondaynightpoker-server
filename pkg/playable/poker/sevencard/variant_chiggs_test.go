package sevencard

import (
	"testing"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// drainLogChannel starts a goroutine to drain log messages
func drainLogChannel(game *Game) {
	go func() {
		//nolint:revive // intentionally empty to drain channel
		for range game.logChan {
		}
	}()
}

func TestChiggs_Name(t *testing.T) {
	c := &Chiggs{}
	assert.Equal(t, "7 Card Chiggs", c.Name())
}

func TestChiggs_AllFoursAreWild(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Test 4 of clubs (mushroom) is wild
	card4c := deck.CardFromString("4c")
	chiggs.ParticipantReceivedCard(game, p(1), card4c)
	a.True(card4c.IsWild, "4 of clubs should be wild")
	a.True(card4c.IsBitSet(isMushroom), "4 of clubs should be marked as mushroom")

	// Test 4 of spades (antidote) is wild
	card4s := deck.CardFromString("4s")
	chiggs.ParticipantReceivedCard(game, p(2), card4s)
	a.True(card4s.IsWild, "4 of spades should be wild")
	a.True(card4s.IsBitSet(isAntidote), "4 of spades should be marked as antidote")

	// Test 4 of diamonds (antidote) is wild
	card4d := deck.CardFromString("4d")
	chiggs.ParticipantReceivedCard(game, p(3), card4d)
	a.True(card4d.IsWild, "4 of diamonds should be wild")
	a.True(card4d.IsBitSet(isAntidote), "4 of diamonds should be marked as antidote")

	// Test 4 of hearts (antidote) is wild
	card4h := deck.CardFromString("4h")
	chiggs.ParticipantReceivedCard(game, p(1), card4h)
	a.True(card4h.IsWild, "4 of hearts should be wild")
	a.True(card4h.IsBitSet(isAntidote), "4 of hearts should be marked as antidote")

	// Test non-4 is not wild
	card5c := deck.CardFromString("5c")
	chiggs.ParticipantReceivedCard(game, p(1), card5c)
	a.False(card5c.IsWild, "5 of clubs should not be wild")
}

func TestChiggs_MushroomAntidoteIdentification(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// 4 of clubs is mushroom
	card4c := deck.CardFromString("4c")
	chiggs.ParticipantReceivedCard(game, p(1), card4c)
	a.True(card4c.IsBitSet(isMushroom), "4c should be mushroom")
	a.False(card4c.IsBitSet(isAntidote), "4c should not be antidote")

	// 4 of spades is antidote
	card4s := deck.CardFromString("4s")
	chiggs.ParticipantReceivedCard(game, p(1), card4s)
	a.False(card4s.IsBitSet(isMushroom), "4s should not be mushroom")
	a.True(card4s.IsBitSet(isAntidote), "4s should be antidote")

	// 4 of diamonds is antidote
	card4d := deck.CardFromString("4d")
	chiggs.ParticipantReceivedCard(game, p(1), card4d)
	a.False(card4d.IsBitSet(isMushroom), "4d should not be mushroom")
	a.True(card4d.IsBitSet(isAntidote), "4d should be antidote")

	// 4 of hearts is antidote
	card4h := deck.CardFromString("4h")
	chiggs.ParticipantReceivedCard(game, p(1), card4h)
	a.False(card4h.IsBitSet(isMushroom), "4h should not be mushroom")
	a.True(card4h.IsBitSet(isAntidote), "4h should be antidote")
}

func TestChiggs_GetNeighbors(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, opts)

	// Player 1 (index 0): neighbors are 4 (left) and 2 (right)
	neighbors := chiggs.getNeighbors(game, 1)
	a.Len(neighbors, 2)
	a.Contains(neighbors, int64(4)) // left wrap-around
	a.Contains(neighbors, int64(2)) // right

	// Player 2 (index 1): neighbors are 1 (left) and 3 (right)
	neighbors = chiggs.getNeighbors(game, 2)
	a.Len(neighbors, 2)
	a.Contains(neighbors, int64(1))
	a.Contains(neighbors, int64(3))

	// Player 4 (index 3): neighbors are 3 (left) and 1 (right, wrap-around)
	neighbors = chiggs.getNeighbors(game, 4)
	a.Len(neighbors, 2)
	a.Contains(neighbors, int64(3))
	a.Contains(neighbors, int64(1))
}

func TestChiggs_GetNeighbors_TwoPlayers(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)

	// With 2 players, each is the other's only neighbor
	neighbors := chiggs.getNeighbors(game, 1)
	a.Len(neighbors, 1)
	a.Equal(int64(2), neighbors[0])

	neighbors = chiggs.getNeighbors(game, 2)
	a.Len(neighbors, 1)
	a.Equal(int64(1), neighbors[0])
}

func TestChiggs_FaceUpMushroomTriggersEvent(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Give player 2 a face-up mushroom
	card4c := deck.CardFromString("4c")
	card4c.SetBit(faceUp)
	p(2).hand.AddCard(card4c)

	// Drain log channel
	drainLogChannel(game)

	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Mushroom holder should be player 2
	a.Equal(int64(2), chiggs.mushroomHolderID, "mushroom holder should be player 2")

	// Neighbors (1 and 3) should be folded since they have no antidotes
	a.True(p(1).didFold, "player 1 should fold without antidote")
	a.True(p(3).didFold, "player 3 should fold without antidote")

	// Mushroom phase should be complete since all neighbors auto-folded (no pending responses)
	a.False(chiggs.mushroomActive, "mushroom phase should complete when all neighbors auto-fold")
}

func TestChiggs_FaceDownMushroomCanBeFlipped(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Give player 2 a face-down mushroom
	card4c := deck.CardFromString("4c")
	// No faceUp bit set = face-down
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Should be tracked as having face-down mushroom
	a.NotNil(chiggs.playersWithFaceDownMushroom[2], "player 2 should have face-down mushroom")

	// Should have flip action available
	actions := chiggs.GetVariantActions(game, p(2))
	a.Contains(actions, ActionFlipMushroom, "player 2 should be able to flip mushroom")

	// Mushroom should not be active yet
	a.False(chiggs.mushroomActive, "mushroom should not be active yet")
}

func TestChiggs_NeighborWithoutAntidoteAutoFolds(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 2 a face-up mushroom
	card4c := deck.CardFromString("4c")
	card4c.SetBit(faceUp)
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Neighbors without antidotes should fold
	a.True(p(1).didFold, "player 1 should auto-fold")
	a.True(p(3).didFold, "player 3 should auto-fold")
	a.False(p(2).didFold, "player 2 should not fold")
}

func TestChiggs_NeighborWithAntidoteMustRespond(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 1 an antidote first
	card4s := deck.CardFromString("4s")
	p(1).hand.AddCard(card4s)
	chiggs.ParticipantReceivedCard(game, p(1), card4s)

	// Give player 2 a face-up mushroom
	card4c := deck.CardFromString("4c")
	card4c.SetBit(faceUp)
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Player 1 should have pending response (has antidote)
	a.True(chiggs.pendingResponses[1], "player 1 should have pending response")
	a.False(p(1).didFold, "player 1 should not fold yet")

	// Player 3 should auto-fold (no antidote)
	a.True(p(3).didFold, "player 3 should auto-fold")

	// Player 1 should have play-antidote action
	actions := chiggs.GetVariantActions(game, p(1))
	a.Contains(actions, ActionPlayAntidote, "player 1 should have play-antidote action")
}

func TestChiggs_AntidotePlayDiscardsCard(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 1 an antidote
	card4s := deck.CardFromString("4s")
	p(1).hand.AddCard(card4s)
	chiggs.ParticipantReceivedCard(game, p(1), card4s)

	// Give player 2 a face-up mushroom
	card4c := deck.CardFromString("4c")
	card4c.SetBit(faceUp)
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Player 1 plays antidote
	handled, err := chiggs.HandleVariantAction(game, p(1), ActionPlayAntidote)
	a.NoError(err)
	a.True(handled)

	// Antidote should be marked as discarded
	a.True(card4s.IsBitSet(wasDiscarded), "antidote should be marked as discarded")
	a.False(card4s.IsWild, "discarded antidote should not be wild")

	// Player 1 should not have pending response anymore
	a.False(chiggs.pendingResponses[1], "player 1 should not have pending response")
	a.False(p(1).didFold, "player 1 should not fold")

	// Mushroom phase should be complete
	a.False(chiggs.mushroomActive, "mushroom phase should be complete")
}

func TestChiggs_FlippedMushroomIsDiscarded(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 2 a face-down mushroom
	card4c := deck.CardFromString("4c")
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Flip the mushroom
	handled, err := chiggs.HandleVariantAction(game, p(2), ActionFlipMushroom)
	a.NoError(err)
	a.True(handled)

	// Mushroom should be marked as discarded
	a.True(card4c.IsBitSet(wasDiscarded), "flipped mushroom should be marked as discarded")
}

func TestChiggs_FaceUpMushroomNotDiscarded(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 2 a face-up mushroom (dealt, not flipped)
	card4c := deck.CardFromString("4c")
	card4c.SetBit(faceUp)
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Mushroom event should have triggered
	a.Equal(int64(2), chiggs.mushroomHolderID, "mushroom event should have triggered")

	// But the mushroom should NOT be discarded - player keeps it as a wild
	a.False(card4c.IsBitSet(wasDiscarded), "face-up dealt mushroom should NOT be discarded")
	a.True(card4c.IsWild, "face-up dealt mushroom should remain wild")
}

func TestChiggs_UnflippedMushroomStaysWild(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Give player 2 a face-down mushroom
	card4c := deck.CardFromString("4c")
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Don't flip it - it should remain wild
	a.True(card4c.IsWild, "unflipped mushroom should remain wild")
	a.False(card4c.IsBitSet(wasDiscarded), "unflipped mushroom should not be discarded")
}

func TestChiggs_Start(t *testing.T) {
	chiggs := &Chiggs{
		mushroomActive:              true,
		mushroomHolderID:            5,
		pendingResponses:            map[int64]bool{1: true},
		playersWithFaceDownMushroom: map[int64]*deck.Card{2: {}},
		lockedMushrooms:             map[int64]bool{2: true},
	}

	chiggs.Start()

	a := assert.New(t)
	a.False(chiggs.mushroomActive)
	a.Equal(int64(0), chiggs.mushroomHolderID)
	a.Empty(chiggs.pendingResponses)
	a.Empty(chiggs.playersWithFaceDownMushroom)
	a.Empty(chiggs.lockedMushrooms)
}

func TestChiggs_IsVariantPhasePending(t *testing.T) {
	a := assert.New(t)
	chiggs := &Chiggs{}
	chiggs.Start()

	// Not pending when mushroom not active
	a.False(chiggs.IsVariantPhasePending())

	// Not pending when active but no responses needed
	chiggs.mushroomActive = true
	a.False(chiggs.IsVariantPhasePending())

	// Pending when active and responses needed
	chiggs.pendingResponses[1] = true
	a.True(chiggs.IsVariantPhasePending())
}

func TestChiggs_GetVariantState(t *testing.T) {
	a := assert.New(t)
	chiggs := &Chiggs{}
	chiggs.Start()

	chiggs.mushroomActive = true
	chiggs.mushroomHolderID = 5

	state := chiggs.GetVariantState().(*ChiggsState)
	a.True(state.MushroomActive)
	a.Equal(int64(5), state.MushroomHolderID)
}

func TestChiggs_BettingActionsSuppressedDuringMushroomPhase(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	a.NoError(game.Start())

	// Drain log channel
	drainLogChannel(game)

	p := createParticipantGetter(game)

	// Simulate mushroom phase pending
	chiggs.mushroomActive = true
	chiggs.pendingResponses[1] = true

	// Betting actions should be suppressed
	actions := game.getActionsForParticipant(p(1))
	a.Empty(actions, "betting actions should be suppressed during mushroom phase")

	// Future actions should also be suppressed
	futureActions := game.getFutureActionsForParticipant(p(2))
	a.Empty(futureActions, "future betting actions should be suppressed during mushroom phase")
}

func TestChiggs_MultipleAntidotesAutoSelectsFirst(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 1 two antidotes
	card4s := deck.CardFromString("4s")
	card4h := deck.CardFromString("4h")
	p(1).hand.AddCard(card4s)
	p(1).hand.AddCard(card4h)
	chiggs.ParticipantReceivedCard(game, p(1), card4s)
	chiggs.ParticipantReceivedCard(game, p(1), card4h)

	// Give player 2 a face-up mushroom
	card4c := deck.CardFromString("4c")
	card4c.SetBit(faceUp)
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Player 1 plays antidote
	handled, err := chiggs.HandleVariantAction(game, p(1), ActionPlayAntidote)
	a.NoError(err)
	a.True(handled)

	// First antidote (4s) should be discarded
	a.True(card4s.IsBitSet(wasDiscarded), "first antidote should be discarded")
	// Second antidote (4h) should still be available
	a.False(card4h.IsBitSet(wasDiscarded), "second antidote should not be discarded")
}

func TestChiggs_Integration_FullGame(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, err := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	a.NoError(err)

	// Set up a controlled deck:
	// First 3 cards (hole cards for each player)
	// Second 3 cards (hole cards)
	// Third 3 cards (face-up, including a mushroom for player 2)
	game.deck.Cards = deck.CardsFromString(
		// Hole cards round 1: player 1, 2, 3
		"2c,3c,4s," + // player 3 gets 4s (antidote)
			// Hole cards round 2
			"5c,6c,7c," +
			// Face-up round 3
			"8c,4c,9c", // player 2 gets 4c (mushroom) face-up
	)

	// Drain log channel
	drainLogChannel(game)

	a.NoError(game.Start())

	p := createParticipantGetter(game)

	// After start, player 2 has face-up mushroom
	// Player 1 should fold (no antidote)
	// Player 3 should have pending antidote response
	a.True(p(1).didFold, "player 1 should fold - no antidote")
	a.False(p(2).didFold, "player 2 should not fold - has mushroom")
	a.False(p(3).didFold, "player 3 should not fold yet - has antidote")

	// Check mushroom phase is pending
	a.True(chiggs.IsVariantPhasePending())

	// Player 3 should have antidote action
	actions := chiggs.GetVariantActions(game, p(3))
	a.Contains(actions, ActionPlayAntidote)

	// Player 3 plays antidote
	resp, updated, err := game.Action(3, &playable.PayloadIn{Action: "play-antidote"})
	a.NoError(err)
	a.True(updated)
	a.NotNil(resp)

	// Mushroom phase should be complete
	a.False(chiggs.IsVariantPhasePending())
	a.False(p(3).didFold, "player 3 should not fold after playing antidote")
}

func TestChiggs_HandleVariantAction_UnhandledAction(t *testing.T) {
	a := assert.New(t)
	chiggs := &Chiggs{}
	chiggs.Start()

	opts := DefaultOptions()
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// Try an unknown action
	handled, err := chiggs.HandleVariantAction(game, p(1), ActionCheck)
	a.NoError(err)
	a.False(handled, "ActionCheck should not be handled by variant")
}

func TestChiggs_FlipMushroom_NoMushroomToFlip(t *testing.T) {
	a := assert.New(t)
	chiggs := &Chiggs{}
	chiggs.Start()

	opts := DefaultOptions()
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// Player has no mushroom to flip
	handled, err := chiggs.HandleVariantAction(game, p(1), ActionFlipMushroom)
	a.NoError(err)
	a.False(handled, "should not handle flip when no mushroom")
}

func TestChiggs_PlayAntidote_NotPending(t *testing.T) {
	a := assert.New(t)
	chiggs := &Chiggs{}
	chiggs.Start()

	opts := DefaultOptions()
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// Player has no pending response
	handled, err := chiggs.HandleVariantAction(game, p(1), ActionPlayAntidote)
	a.NoError(err)
	a.False(handled, "should not handle play-antidote when not pending")
}

func TestChiggs_FoldedPlayersExcludedFromMushroom(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Player 1 already folded
	p(1).didFold = true

	// Give player 2 a face-up mushroom
	card4c := deck.CardFromString("4c")
	card4c.SetBit(faceUp)
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Player 1 was already folded, so no change
	a.True(p(1).didFold)
	// Player 3 (other neighbor) should fold
	a.True(p(3).didFold)
}

func TestChiggs_MushroomFoldsTrackedInVariantState(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 2 a face-up mushroom
	card4c := deck.CardFromString("4c")
	card4c.SetBit(faceUp)
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Neighbors (1 and 3) should fold since they have no antidotes
	a.True(p(1).didFold, "player 1 should fold without antidote")
	a.True(p(3).didFold, "player 3 should fold without antidote")

	// Mushroom folds should be tracked in variant state
	state := chiggs.GetVariantState().(*ChiggsState)
	a.Len(state.MushroomFolds, 2, "should track 2 mushroom folds")

	// Verify the folded players are tracked
	foldedPlayerIDs := make(map[int64]bool)
	for _, fold := range state.MushroomFolds {
		foldedPlayerIDs[fold.PlayerID] = true
	}
	a.True(foldedPlayerIDs[1], "player 1 should be in mushroom folds")
	a.True(foldedPlayerIDs[3], "player 3 should be in mushroom folds")
}

func TestChiggs_ClearMushroomFolds(t *testing.T) {
	a := assert.New(t)
	chiggs := &Chiggs{}
	chiggs.Start()

	// Set some mushroom folds
	chiggs.lastMushroomFolds = []*mushroomFoldInfo{
		{PlayerID: 1},
		{PlayerID: 3},
	}

	// Clear them
	chiggs.ClearMushroomFolds()

	a.Nil(chiggs.lastMushroomFolds, "mushroom folds should be cleared")

	// Verify variant state shows no folds
	state := chiggs.GetVariantState().(*ChiggsState)
	a.Nil(state.MushroomFolds, "variant state should show no mushroom folds")
}

func TestChiggs_MushroomLockedAfterBet(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Give player 2 a face-down mushroom
	card4c := deck.CardFromString("4c")
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Should be tracked as having face-down mushroom
	a.NotNil(chiggs.playersWithFaceDownMushroom[2], "player 2 should have face-down mushroom")

	// Before any bet, should have flip action
	actions := chiggs.GetVariantActions(game, p(2))
	a.Contains(actions, ActionFlipMushroom, "player 2 should be able to flip mushroom before bet")

	// Simulate a bet being placed
	chiggs.OnBetPlaced(game)

	// After bet, mushroom should be locked
	a.True(chiggs.lockedMushrooms[2], "player 2's mushroom should be locked after bet")

	// Should NOT have flip action anymore
	actions = chiggs.GetVariantActions(game, p(2))
	a.NotContains(actions, ActionFlipMushroom, "player 2 should NOT be able to flip mushroom after bet")

	// Also verify GetVariantStateForPlayer returns canFlip = false
	state := chiggs.GetVariantStateForPlayer(2)
	a.False(state.CanFlipMushroom, "canFlipMushroom should be false after bet")
}

func TestChiggs_MushroomNotLockedAfterCheck(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Give player 2 a face-down mushroom
	card4c := deck.CardFromString("4c")
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Simulate checks (no bet placed, so OnBetPlaced is never called)
	// Advance decision count to show someone checked, but stay within bounds
	game.decisionCount = 1

	// Mushroom should NOT be locked (check/fold don't lock)
	a.False(chiggs.lockedMushrooms[2], "player 2's mushroom should NOT be locked after check")

	// Should still have flip action (round not over yet)
	actions := chiggs.GetVariantActions(game, p(2))
	a.Contains(actions, ActionFlipMushroom, "player 2 should still be able to flip mushroom after check")

	// Also verify GetVariantStateForPlayer returns canFlip = true
	state := chiggs.GetVariantStateForPlayer(2)
	a.True(state.CanFlipMushroom, "canFlipMushroom should be true after check")
}

func TestChiggs_MushroomLockedPermanentlyAcrossRounds(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Simulate round 3 where player 2 gets a face-down mushroom
	game.round = 3
	card4c := deck.CardFromString("4c")
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// No bet on round 3, mushroom is still flippable
	a.False(chiggs.lockedMushrooms[2], "mushroom should not be locked before bet")
	actions := chiggs.GetVariantActions(game, p(2))
	a.Contains(actions, ActionFlipMushroom, "should be able to flip before bet")

	// Advance to round 4
	game.round = 4
	game.decisionCount = 0

	// Someone places a bet on round 4
	chiggs.OnBetPlaced(game)

	// Mushroom from round 3 should now be permanently locked
	a.True(chiggs.lockedMushrooms[2], "mushroom should be locked after bet in round 4")

	// Even on final round, mushroom is still locked
	game.round = finalBettingRound
	game.decisionCount = 0

	actions = chiggs.GetVariantActions(game, p(2))
	a.NotContains(actions, ActionFlipMushroom, "mushroom should remain locked for rest of game")

	state := chiggs.GetVariantStateForPlayer(2)
	a.False(state.CanFlipMushroom, "canFlipMushroom should be false permanently after bet")
}

func TestChiggs_MushroomCannotFlipWhenRoundOver(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	chiggs := &Chiggs{}
	opts.Variant = chiggs
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Give player 2 a face-down mushroom
	card4c := deck.CardFromString("4c")
	p(2).hand.AddCard(card4c)
	chiggs.ParticipantReceivedCard(game, p(2), card4c)

	// Mushroom is not locked yet
	a.False(chiggs.lockedMushrooms[2], "mushroom should not be locked")

	// During betting round, should be able to flip
	game.round = 3
	game.decisionCount = 0 // Current turn is set (someone still to act)
	actions := chiggs.GetVariantActions(game, p(2))
	a.Contains(actions, ActionFlipMushroom, "should be able to flip during betting")

	// Simulate round being over (no current turn)
	// Set decisionCount to >= len(playerIDs) so getCurrentTurn() returns nil
	game.decisionCount = len(game.playerIDs)

	// When round is over (getCurrentTurn() == nil), cannot flip
	actions = chiggs.GetVariantActions(game, p(2))
	a.NotContains(actions, ActionFlipMushroom, "should NOT be able to flip when round is over")

	// Also verify GetVariantStateForPlayer returns canFlip = false
	state := chiggs.GetVariantStateForPlayer(2)
	a.False(state.CanFlipMushroom, "canFlipMushroom should be false when round is over")
}
