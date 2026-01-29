package sevencard

import (
	"testing"

	"mondaynightpoker-server/pkg/deck"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCouponsAndClippings_Name(t *testing.T) {
	cc := &CouponsAndClippings{}
	assert.Equal(t, "Coupons and Clippings", cc.Name())
}

func TestCouponsAndClippings_Start(t *testing.T) {
	cc := &CouponsAndClippings{
		faceUpRankCounts:      map[int]int{3: 2, 5: 1},
		currentWildRank:       3,
		bogoPlayerID:          42,
		nailClippingPlayerIDs: []int64{1, 2},
		lastDealRound:         round(5),
	}

	cc.Start()

	a := assert.New(t)
	a.NotNil(cc.faceUpRankCounts)
	a.Empty(cc.faceUpRankCounts)
	a.Equal(0, cc.currentWildRank)
	a.Equal(int64(0), cc.bogoPlayerID)
	a.Nil(cc.nailClippingPlayerIDs)
	a.Equal(round(0), cc.lastDealRound)
}

func TestCouponsAndClippings_NoWildsAtStart(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// Any card dealt should not be wild (no wilds at start)
	cardAc := deck.CardFromString("14c") // Ace of clubs
	cc.ParticipantReceivedCard(game, p(1), cardAc)
	a.False(cardAc.IsWild, "Ace should not be wild at start")

	card3c := deck.CardFromString("3c")
	cc.ParticipantReceivedCard(game, p(1), card3c)
	a.False(card3c.IsWild, "3 should not be wild at start (unlike old rules)")

	card2c := deck.CardFromString("2c")
	cc.ParticipantReceivedCard(game, p(1), card2c)
	a.False(card2c.IsWild, "2 should not be wild at start")
}

func TestCouponsAndClippings_FirstFaceUpCardNotWild(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// First face-up 5 should NOT be wild
	card5c := deck.CardFromString("5c")
	card5c.SetBit(faceUp)
	p(1).hand.AddCard(card5c)
	cc.ParticipantReceivedCard(game, p(1), card5c)

	a.False(card5c.IsWild, "First face-up card of a rank should NOT be wild")
	a.Equal(0, cc.currentWildRank, "No wild rank should be set yet")
}

func TestCouponsAndClippings_SecondFaceUpCardTriggersBogo(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// First face-up 5
	card5c := deck.CardFromString("5c")
	card5c.SetBit(faceUp)
	p(1).hand.AddCard(card5c)
	cc.ParticipantReceivedCard(game, p(1), card5c)

	a.False(card5c.IsWild, "First 5 should not be wild")

	// Second face-up 5 triggers BOGO
	card5d := deck.CardFromString("5d")
	card5d.SetBit(faceUp)
	p(2).hand.AddCard(card5d)
	cc.ParticipantReceivedCard(game, p(2), card5d)

	a.True(card5c.IsWild, "First 5 should now be wild after BOGO")
	a.True(card5d.IsWild, "Second 5 should be wild after BOGO")
	a.Equal(5, cc.currentWildRank, "Current wild rank should be 5")
	a.Equal(int64(2), cc.bogoPlayerID, "BOGO player should be player 2")
}

func TestCouponsAndClippings_BogoAffectsAllHands(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// Give each player a 7 (face-down)
	card7c := deck.CardFromString("7c")
	p(1).hand.AddCard(card7c)
	cc.ParticipantReceivedCard(game, p(1), card7c)

	card7d := deck.CardFromString("7d")
	p(2).hand.AddCard(card7d)
	cc.ParticipantReceivedCard(game, p(2), card7d)

	card7h := deck.CardFromString("7h")
	p(3).hand.AddCard(card7h)
	cc.ParticipantReceivedCard(game, p(3), card7h)

	card7s := deck.CardFromString("7s")
	p(4).hand.AddCard(card7s)
	cc.ParticipantReceivedCard(game, p(4), card7s)

	// None should be wild yet
	a.False(card7c.IsWild)
	a.False(card7d.IsWild)
	a.False(card7h.IsWild)
	a.False(card7s.IsWild)

	// Now deal two face-up 7s to trigger BOGO
	cardFaceUp7a := deck.CardFromString("7c")
	cardFaceUp7a.SetBit(faceUp)
	p(1).hand.AddCard(cardFaceUp7a)
	cc.ParticipantReceivedCard(game, p(1), cardFaceUp7a)

	cardFaceUp7b := deck.CardFromString("7d")
	cardFaceUp7b.SetBit(faceUp)
	p(2).hand.AddCard(cardFaceUp7b)
	cc.ParticipantReceivedCard(game, p(2), cardFaceUp7b)

	// ALL 7s should now be wild (including the face-down ones)
	a.True(card7c.IsWild, "Player 1's face-down 7 should be wild")
	a.True(card7d.IsWild, "Player 2's face-down 7 should be wild")
	a.True(card7h.IsWild, "Player 3's face-down 7 should be wild")
	a.True(card7s.IsWild, "Player 4's face-down 7 should be wild")
	a.True(cardFaceUp7a.IsWild, "Face-up 7 should be wild")
	a.True(cardFaceUp7b.IsWild, "Face-up 7 should be wild")
}

func TestCouponsAndClippings_NewBogoReplacesOldWilds(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// First, give player a 5 and trigger BOGO on 5s
	card5c := deck.CardFromString("5c")
	p(1).hand.AddCard(card5c)

	card5FaceUp1 := deck.CardFromString("5d")
	card5FaceUp1.SetBit(faceUp)
	p(1).hand.AddCard(card5FaceUp1)
	cc.ParticipantReceivedCard(game, p(1), card5FaceUp1)

	card5FaceUp2 := deck.CardFromString("5h")
	card5FaceUp2.SetBit(faceUp)
	p(2).hand.AddCard(card5FaceUp2)
	cc.ParticipantReceivedCard(game, p(2), card5FaceUp2)

	// 5s should be wild
	a.True(card5FaceUp1.IsWild)
	a.True(card5FaceUp2.IsWild)
	a.Equal(5, cc.currentWildRank)

	// Now trigger BOGO on 8s
	card8a := deck.CardFromString("8c")
	card8a.SetBit(faceUp)
	p(1).hand.AddCard(card8a)
	cc.ParticipantReceivedCard(game, p(1), card8a)

	card8b := deck.CardFromString("8d")
	card8b.SetBit(faceUp)
	p(2).hand.AddCard(card8b)
	cc.ParticipantReceivedCard(game, p(2), card8b)

	// Now 8s should be wild, and 5s should NOT be wild
	a.False(card5FaceUp1.IsWild, "5 should no longer be wild after new BOGO")
	a.False(card5FaceUp2.IsWild, "5 should no longer be wild after new BOGO")
	a.True(card8a.IsWild, "8 should be wild after BOGO")
	a.True(card8b.IsWild, "8 should be wild after BOGO")
	a.Equal(8, cc.currentWildRank)
}

func TestCouponsAndClippings_FaceDownCardDoesNotTriggerBogo(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// First face-up 5
	card5FaceUp := deck.CardFromString("5c")
	card5FaceUp.SetBit(faceUp)
	p(1).hand.AddCard(card5FaceUp)
	cc.ParticipantReceivedCard(game, p(1), card5FaceUp)

	// Second 5 is face-down - should NOT trigger BOGO
	card5FaceDown := deck.CardFromString("5d")
	// No faceUp bit
	p(2).hand.AddCard(card5FaceDown)
	cc.ParticipantReceivedCard(game, p(2), card5FaceDown)

	a.False(card5FaceUp.IsWild, "Face-up 5 should not be wild")
	a.False(card5FaceDown.IsWild, "Face-down 5 should not be wild")
	a.Equal(0, cc.currentWildRank, "No wild rank should be set")
	a.Equal(1, cc.faceUpRankCounts[5], "Only one face-up 5 should be counted")
}

func TestCouponsAndClippings_FaceDownCardBecomesWildIfMatchingBogoRank(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// Trigger BOGO on 5s
	card5FaceUp1 := deck.CardFromString("5c")
	card5FaceUp1.SetBit(faceUp)
	p(1).hand.AddCard(card5FaceUp1)
	cc.ParticipantReceivedCard(game, p(1), card5FaceUp1)

	card5FaceUp2 := deck.CardFromString("5d")
	card5FaceUp2.SetBit(faceUp)
	p(2).hand.AddCard(card5FaceUp2)
	cc.ParticipantReceivedCard(game, p(2), card5FaceUp2)

	a.Equal(5, cc.currentWildRank)

	// Now deal a face-down 5 - it should become wild
	card5FaceDown := deck.CardFromString("5h")
	p(3).hand.AddCard(card5FaceDown)
	cc.ParticipantReceivedCard(game, p(3), card5FaceDown)

	a.True(card5FaceDown.IsWild, "Face-down 5 should be wild since 5s are the BOGO rank")
}

func TestCouponsAndClippings_ThirdFaceUpCardNoAdditionalEffect(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// Trigger BOGO on 5s with first two face-up 5s
	card5a := deck.CardFromString("5c")
	card5a.SetBit(faceUp)
	p(1).hand.AddCard(card5a)
	cc.ParticipantReceivedCard(game, p(1), card5a)

	card5b := deck.CardFromString("5d")
	card5b.SetBit(faceUp)
	p(2).hand.AddCard(card5b)
	cc.ParticipantReceivedCard(game, p(2), card5b)

	a.Equal(int64(2), cc.bogoPlayerID, "Player 2 triggered BOGO")

	// Third face-up 5 - no additional BOGO trigger
	game.round++ // New round to reset splash state
	card5c := deck.CardFromString("5h")
	card5c.SetBit(faceUp)
	p(3).hand.AddCard(card5c)
	cc.ParticipantReceivedCard(game, p(3), card5c)

	// Card should still be wild (matches current wild rank)
	a.True(card5c.IsWild, "Third face-up 5 should be wild")
	a.Equal(5, cc.currentWildRank, "Wild rank should still be 5")
	// bogoPlayerID should be reset due to new round
	a.Equal(int64(0), cc.bogoPlayerID, "BOGO player should be reset for new round")
}

func TestCouponsAndClippings_NailClippingRefundsAnte(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	opts.Ante = 25
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	initialPot := game.pot
	initialBalance := p(1).balance

	// Face-up 10 should refund ante
	card10c := deck.CardFromString("10c")
	card10c.SetBit(faceUp)
	p(1).hand.AddCard(card10c)
	cc.ParticipantReceivedCard(game, p(1), card10c)

	a.Equal(initialPot-25, game.pot, "Pot should be reduced by ante")
	a.Equal(initialBalance+25, p(1).balance, "Player balance should increase by ante")
	a.Contains(cc.nailClippingPlayerIDs, int64(1), "Player 1 should be in nail clipping list")
}

func TestCouponsAndClippings_FaceDownTenNoRefund(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	opts.Ante = 25
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	initialPot := game.pot
	initialBalance := p(1).balance

	// Face-down 10 should NOT refund ante
	card10c := deck.CardFromString("10c")
	// No faceUp bit
	p(1).hand.AddCard(card10c)
	cc.ParticipantReceivedCard(game, p(1), card10c)

	a.Equal(initialPot, game.pot, "Pot should not change")
	a.Equal(initialBalance, p(1).balance, "Player balance should not change")
	a.Empty(cc.nailClippingPlayerIDs, "No players should be in nail clipping list")
}

func TestCouponsAndClippings_InsufficientPotNoRefund(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	opts.Ante = 25
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// Drain the pot
	game.pot = 10 // Less than ante

	initialBalance := p(1).balance

	// Face-up 10 should NOT refund ante (pot too small)
	card10c := deck.CardFromString("10c")
	card10c.SetBit(faceUp)
	p(1).hand.AddCard(card10c)
	cc.ParticipantReceivedCard(game, p(1), card10c)

	a.Equal(10, game.pot, "Pot should not change")
	a.Equal(initialBalance, p(1).balance, "Player balance should not change")
	a.Empty(cc.nailClippingPlayerIDs, "Player should not be in nail clipping list")
}

func TestCouponsAndClippings_MultipleTensMultipleRefunds(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	opts.Ante = 25
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	initialPot := game.pot
	initialBalance1 := p(1).balance
	initialBalance2 := p(2).balance

	// First face-up 10
	card10c := deck.CardFromString("10c")
	card10c.SetBit(faceUp)
	p(1).hand.AddCard(card10c)
	cc.ParticipantReceivedCard(game, p(1), card10c)

	// Second face-up 10 (also triggers BOGO on 10s)
	card10d := deck.CardFromString("10d")
	card10d.SetBit(faceUp)
	p(2).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(2), card10d)

	a.Equal(initialPot-50, game.pot, "Pot should be reduced by 2 antes")
	a.Equal(initialBalance1+25, p(1).balance, "Player 1 balance should increase by ante")
	a.Equal(initialBalance2+25, p(2).balance, "Player 2 balance should increase by ante")
	a.Contains(cc.nailClippingPlayerIDs, int64(1))
	a.Contains(cc.nailClippingPlayerIDs, int64(2))
	a.Len(cc.nailClippingPlayerIDs, 2)
}

func TestCouponsAndClippings_TenCanAlsoTriggerBogo(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	opts.Ante = 25
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// First face-up 10
	card10c := deck.CardFromString("10c")
	card10c.SetBit(faceUp)
	p(1).hand.AddCard(card10c)
	cc.ParticipantReceivedCard(game, p(1), card10c)

	a.False(card10c.IsWild, "First 10 should not be wild yet")

	// Second face-up 10 - should trigger both nail clipping AND BOGO
	card10d := deck.CardFromString("10d")
	card10d.SetBit(faceUp)
	p(2).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(2), card10d)

	// Both 10s should now be wild
	a.True(card10c.IsWild, "First 10 should be wild after BOGO")
	a.True(card10d.IsWild, "Second 10 should be wild after BOGO")
	a.Equal(10, cc.currentWildRank, "10s should be the wild rank")

	// Both players should have gotten refunds
	a.Len(cc.nailClippingPlayerIDs, 2)
}

func TestCouponsAndClippings_SplashStateResetsBetweenRounds(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	opts.Ante = 25
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// Trigger BOGO in round 1
	card5a := deck.CardFromString("5c")
	card5a.SetBit(faceUp)
	p(1).hand.AddCard(card5a)
	cc.ParticipantReceivedCard(game, p(1), card5a)

	card5b := deck.CardFromString("5d")
	card5b.SetBit(faceUp)
	p(2).hand.AddCard(card5b)
	cc.ParticipantReceivedCard(game, p(2), card5b)

	a.Equal(int64(2), cc.bogoPlayerID)

	// Get nail clipping too
	card10c := deck.CardFromString("10c")
	card10c.SetBit(faceUp)
	p(1).hand.AddCard(card10c)
	cc.ParticipantReceivedCard(game, p(1), card10c)

	a.Contains(cc.nailClippingPlayerIDs, int64(1))

	// Advance to next round
	game.round++

	// Deal a card to trigger splash reset
	card6c := deck.CardFromString("6c")
	card6c.SetBit(faceUp)
	p(1).hand.AddCard(card6c)
	cc.ParticipantReceivedCard(game, p(1), card6c)

	// Splash state should be reset
	a.Equal(int64(0), cc.bogoPlayerID, "BOGO player should be reset")
	a.Empty(cc.nailClippingPlayerIDs, "Nail clipping players should be reset")

	// But wild rank and face-up counts should persist
	a.Equal(5, cc.currentWildRank, "Wild rank should persist")
	a.Equal(2, cc.faceUpRankCounts[5], "Face-up counts should persist")
}

func TestCouponsAndClippings_WildRankPersistsAcrossRounds(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// Trigger BOGO on 5s in round 1
	card5a := deck.CardFromString("5c")
	card5a.SetBit(faceUp)
	p(1).hand.AddCard(card5a)
	cc.ParticipantReceivedCard(game, p(1), card5a)

	card5b := deck.CardFromString("5d")
	card5b.SetBit(faceUp)
	p(2).hand.AddCard(card5b)
	cc.ParticipantReceivedCard(game, p(2), card5b)

	a.Equal(5, cc.currentWildRank)

	// Advance several rounds
	game.round++
	game.round++
	game.round++

	// Deal another 5 (face-up) - should be wild
	card5c := deck.CardFromString("5h")
	card5c.SetBit(faceUp)
	p(3).hand.AddCard(card5c)
	cc.ParticipantReceivedCard(game, p(3), card5c)

	a.True(card5c.IsWild, "5 should still be wild in later rounds")
	a.Equal(5, cc.currentWildRank, "Wild rank should persist")
}

func TestCouponsAndClippings_FaceUpCountsPersistAcrossRounds(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// Deal first face-up 5 in round 1
	card5a := deck.CardFromString("5c")
	card5a.SetBit(faceUp)
	p(1).hand.AddCard(card5a)
	cc.ParticipantReceivedCard(game, p(1), card5a)

	a.Equal(1, cc.faceUpRankCounts[5])
	a.Equal(0, cc.currentWildRank)

	// Advance to round 2
	game.round++

	// Deal second face-up 5 in round 2 - should trigger BOGO
	card5b := deck.CardFromString("5d")
	card5b.SetBit(faceUp)
	p(2).hand.AddCard(card5b)
	cc.ParticipantReceivedCard(game, p(2), card5b)

	a.Equal(2, cc.faceUpRankCounts[5])
	a.Equal(5, cc.currentWildRank, "BOGO should trigger across rounds")
	a.True(card5a.IsWild)
	a.True(card5b.IsWild)
}

func TestCouponsAndClippings_GetVariantState(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	// Trigger BOGO
	card5a := deck.CardFromString("5c")
	card5a.SetBit(faceUp)
	p(1).hand.AddCard(card5a)
	cc.ParticipantReceivedCard(game, p(1), card5a)

	card5b := deck.CardFromString("5d")
	card5b.SetBit(faceUp)
	p(2).hand.AddCard(card5b)
	cc.ParticipantReceivedCard(game, p(2), card5b)

	// Add nail clipping
	card10c := deck.CardFromString("10c")
	card10c.SetBit(faceUp)
	p(1).hand.AddCard(card10c)
	cc.ParticipantReceivedCard(game, p(1), card10c)

	card10d := deck.CardFromString("10d")
	card10d.SetBit(faceUp)
	p(3).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(3), card10d)

	state := cc.GetVariantState().(*CouponsAndClippingsState)
	a.Equal(10, state.CurrentWildRank, "10s became wild after BOGO")
	a.Equal(int64(3), state.BogoPlayerID, "Player 3 triggered second BOGO")
	a.Contains(state.NailClippingPlayerIDs, int64(1))
	a.Contains(state.NailClippingPlayerIDs, int64(3))
}

func TestCouponsAndClippings_IsVariantPhasePending(t *testing.T) {
	cc := &CouponsAndClippings{}
	// CouponsAndClippings never has pending phases (no player actions needed)
	assert.False(t, cc.IsVariantPhasePending())
}

func TestCouponsAndClippings_GetVariantActions(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// CouponsAndClippings has no variant-specific actions
	actions := cc.GetVariantActions(game, p(1))
	a.Nil(actions, "CouponsAndClippings should have no variant actions")
}

func TestCouponsAndClippings_HandleVariantAction(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// CouponsAndClippings doesn't handle any variant actions
	handled, err := cc.HandleVariantAction(game, p(1), ActionCheck)
	a.NoError(err)
	a.False(handled, "CouponsAndClippings should not handle any variant actions")
}

func TestCouponsAndClippings_NailClippingAndBogoSameCard(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	opts.Ante = 25
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	drainLogChannel(game)

	initialPot := game.pot
	initialBalance1 := p(1).balance
	initialBalance2 := p(2).balance

	// First face-up 10 - gets refund but not wild yet
	card10a := deck.CardFromString("10c")
	card10a.SetBit(faceUp)
	p(1).hand.AddCard(card10a)
	cc.ParticipantReceivedCard(game, p(1), card10a)

	a.Equal(initialPot-25, game.pot)
	a.Equal(initialBalance1+25, p(1).balance)
	a.False(card10a.IsWild)

	// Second face-up 10 - gets refund AND triggers BOGO
	card10b := deck.CardFromString("10d")
	card10b.SetBit(faceUp)
	p(2).hand.AddCard(card10b)
	cc.ParticipantReceivedCard(game, p(2), card10b)

	a.Equal(initialPot-50, game.pot, "Both players got refunds")
	a.Equal(initialBalance2+25, p(2).balance)
	a.True(card10a.IsWild, "First 10 is now wild")
	a.True(card10b.IsWild, "Second 10 is now wild")
	a.Equal(10, cc.currentWildRank)
}
