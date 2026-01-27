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

func TestCouponsAndClippings_ThreesAreWildBeforeExpiration(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// 3 of clubs should be wild
	card3c := deck.CardFromString("3c")
	cc.ParticipantReceivedCard(game, p(1), card3c)
	a.True(card3c.IsWild, "3 of clubs should be wild before expiration")

	// 3 of spades should be wild
	card3s := deck.CardFromString("3s")
	cc.ParticipantReceivedCard(game, p(2), card3s)
	a.True(card3s.IsWild, "3 of spades should be wild before expiration")

	// 3 of diamonds should be wild
	card3d := deck.CardFromString("3d")
	cc.ParticipantReceivedCard(game, p(3), card3d)
	a.True(card3d.IsWild, "3 of diamonds should be wild before expiration")

	// 3 of hearts should be wild
	card3h := deck.CardFromString("3h")
	cc.ParticipantReceivedCard(game, p(1), card3h)
	a.True(card3h.IsWild, "3 of hearts should be wild before expiration")
}

func TestCouponsAndClippings_ThreesExpireWhenTenSpadesDealtFaceUp(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 1 a 3 (wild)
	card3c := deck.CardFromString("3c")
	p(1).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(1), card3c)
	a.True(card3c.IsWild, "3 should be wild initially")

	// Give player 2 a 3 (wild)
	card3d := deck.CardFromString("3d")
	p(2).hand.AddCard(card3d)
	cc.ParticipantReceivedCard(game, p(2), card3d)
	a.True(card3d.IsWild, "3 should be wild initially")

	// Deal 10♠ face-up to player 3
	card10s := deck.CardFromString("10s")
	card10s.SetBit(faceUp)
	p(3).hand.AddCard(card10s)
	cc.ParticipantReceivedCard(game, p(3), card10s)

	// All 3s should now be expired (not wild)
	a.True(cc.couponsExpired, "coupons should be expired after 10♠")
	a.False(card3c.IsWild, "3c should no longer be wild after expiration")
	a.False(card3d.IsWild, "3d should no longer be wild after expiration")
}

func TestCouponsAndClippings_TenSpadesFaceDownDoesNotExpire(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 1 a 3 (wild)
	card3c := deck.CardFromString("3c")
	p(1).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(1), card3c)
	a.True(card3c.IsWild, "3 should be wild initially")

	// Deal 10♠ face-down to player 2
	card10s := deck.CardFromString("10s")
	// No faceUp bit = face-down
	p(2).hand.AddCard(card10s)
	cc.ParticipantReceivedCard(game, p(2), card10s)

	// Coupons should NOT be expired
	a.False(cc.couponsExpired, "coupons should NOT be expired when 10♠ dealt face-down")
	a.True(card3c.IsWild, "3 should still be wild")
}

func TestCouponsAndClippings_OtherTensFaceUpDoNotExpire(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 1 a 3 (wild)
	card3c := deck.CardFromString("3c")
	p(1).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(1), card3c)
	a.True(card3c.IsWild, "3 should be wild initially")

	// Deal 10♥ face-up to player 2
	card10h := deck.CardFromString("10h")
	card10h.SetBit(faceUp)
	p(2).hand.AddCard(card10h)
	cc.ParticipantReceivedCard(game, p(2), card10h)

	// Coupons should NOT be expired
	a.False(cc.couponsExpired, "coupons should NOT be expired by 10♥")
	a.True(card3c.IsWild, "3 should still be wild")

	// Deal 10♣ face-up to player 3
	card10c := deck.CardFromString("10c")
	card10c.SetBit(faceUp)
	p(3).hand.AddCard(card10c)
	cc.ParticipantReceivedCard(game, p(3), card10c)

	// Coupons should still NOT be expired
	a.False(cc.couponsExpired, "coupons should NOT be expired by 10♣")
	a.True(card3c.IsWild, "3 should still be wild")
}

func TestCouponsAndClippings_TwosNotWildByDefault(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// 2 of clubs should NOT be wild
	card2c := deck.CardFromString("2c")
	cc.ParticipantReceivedCard(game, p(1), card2c)
	a.False(card2c.IsWild, "2 of clubs should NOT be wild by default")

	// 2 of spades should NOT be wild
	card2s := deck.CardFromString("2s")
	cc.ParticipantReceivedCard(game, p(2), card2s)
	a.False(card2s.IsWild, "2 of spades should NOT be wild by default")
}

func TestCouponsAndClippings_TwosWildAfterTenDiamondsFaceUp(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 1 a 2 (not wild yet)
	card2c := deck.CardFromString("2c")
	p(1).hand.AddCard(card2c)
	cc.ParticipantReceivedCard(game, p(1), card2c)
	a.False(card2c.IsWild, "2 should NOT be wild before meal comp")

	// Deal 10♦ face-up to player 2
	card10d := deck.CardFromString("10d")
	card10d.SetBit(faceUp)
	p(2).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(2), card10d)

	// Meal comp should be active
	a.True(cc.mealComped, "meal comp should be active after 10♦")

	// The 2 in player 1's hand should now be wild
	a.True(card2c.IsWild, "2 should be wild after meal comp")
}

func TestCouponsAndClippings_TenDiamondsFaceDownDoesNotActivate(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give player 1 a 2
	card2c := deck.CardFromString("2c")
	p(1).hand.AddCard(card2c)
	cc.ParticipantReceivedCard(game, p(1), card2c)
	a.False(card2c.IsWild, "2 should NOT be wild before meal comp")

	// Deal 10♦ face-down to player 2
	card10d := deck.CardFromString("10d")
	// No faceUp bit = face-down
	p(2).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(2), card10d)

	// Meal comp should NOT be active
	a.False(cc.mealComped, "meal comp should NOT be active when 10♦ dealt face-down")
	a.False(card2c.IsWild, "2 should NOT be wild")
}

//nolint:dupl // Similar structure to TestCouponsAndClippings_MealCompAffectsAllPlayers but tests different feature
func TestCouponsAndClippings_ExpirationAffectsAllPlayers(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give each player a 3
	card3c := deck.CardFromString("3c")
	p(1).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(1), card3c)

	card3d := deck.CardFromString("3d")
	p(2).hand.AddCard(card3d)
	cc.ParticipantReceivedCard(game, p(2), card3d)

	card3h := deck.CardFromString("3h")
	p(3).hand.AddCard(card3h)
	cc.ParticipantReceivedCard(game, p(3), card3h)

	card3s := deck.CardFromString("3s")
	p(4).hand.AddCard(card3s)
	cc.ParticipantReceivedCard(game, p(4), card3s)

	// All should be wild
	a.True(card3c.IsWild)
	a.True(card3d.IsWild)
	a.True(card3h.IsWild)
	a.True(card3s.IsWild)

	// Deal 10♠ face-up to player 1
	card10s := deck.CardFromString("10s")
	card10s.SetBit(faceUp)
	p(1).hand.AddCard(card10s)
	cc.ParticipantReceivedCard(game, p(1), card10s)

	// ALL 3s should now be expired
	a.False(card3c.IsWild, "player 1's 3 should expire")
	a.False(card3d.IsWild, "player 2's 3 should expire")
	a.False(card3h.IsWild, "player 3's 3 should expire")
	a.False(card3s.IsWild, "player 4's 3 should expire")
}

//nolint:dupl // Similar structure to TestCouponsAndClippings_ExpirationAffectsAllPlayers but tests different feature
func TestCouponsAndClippings_MealCompAffectsAllPlayers(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give each player a 2
	card2c := deck.CardFromString("2c")
	p(1).hand.AddCard(card2c)
	cc.ParticipantReceivedCard(game, p(1), card2c)

	card2d := deck.CardFromString("2d")
	p(2).hand.AddCard(card2d)
	cc.ParticipantReceivedCard(game, p(2), card2d)

	card2h := deck.CardFromString("2h")
	p(3).hand.AddCard(card2h)
	cc.ParticipantReceivedCard(game, p(3), card2h)

	card2s := deck.CardFromString("2s")
	p(4).hand.AddCard(card2s)
	cc.ParticipantReceivedCard(game, p(4), card2s)

	// None should be wild
	a.False(card2c.IsWild)
	a.False(card2d.IsWild)
	a.False(card2h.IsWild)
	a.False(card2s.IsWild)

	// Deal 10♦ face-up to player 1
	card10d := deck.CardFromString("10d")
	card10d.SetBit(faceUp)
	p(1).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(1), card10d)

	// ALL 2s should now be wild
	a.True(card2c.IsWild, "player 1's 2 should become wild")
	a.True(card2d.IsWild, "player 2's 2 should become wild")
	a.True(card2h.IsWild, "player 3's 2 should become wild")
	a.True(card2s.IsWild, "player 4's 2 should become wild")
}

func TestCouponsAndClippings_ThreesDealtAfterExpirationNotWild(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Deal 10♠ face-up first
	card10s := deck.CardFromString("10s")
	card10s.SetBit(faceUp)
	p(1).hand.AddCard(card10s)
	cc.ParticipantReceivedCard(game, p(1), card10s)

	a.True(cc.couponsExpired, "coupons should be expired")

	// Now deal a 3 - it should NOT be wild
	card3c := deck.CardFromString("3c")
	p(2).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(2), card3c)

	a.False(card3c.IsWild, "3 dealt after expiration should NOT be wild")
}

func TestCouponsAndClippings_TwosDealtAfterMealCompAreWild(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Deal 10♦ face-up first
	card10d := deck.CardFromString("10d")
	card10d.SetBit(faceUp)
	p(1).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(1), card10d)

	a.True(cc.mealComped, "meal comp should be active")

	// Now deal a 2 - it SHOULD be wild
	card2c := deck.CardFromString("2c")
	p(2).hand.AddCard(card2c)
	cc.ParticipantReceivedCard(game, p(2), card2c)

	a.True(card2c.IsWild, "2 dealt after meal comp should be wild")
}

func TestCouponsAndClippings_VariantStateTracksSplashPlayerIds(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Deal a face-up 3 to player 2
	card3c := deck.CardFromString("3c")
	card3c.SetBit(faceUp)
	p(2).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(2), card3c)

	state := cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.CouponPlayerIDs, int64(2), "coupon players should include 2")

	// Advance to next round and deal 10♠ face-up to player 1
	game.round++
	card10s := deck.CardFromString("10s")
	card10s.SetBit(faceUp)
	p(1).hand.AddCard(card10s)
	cc.ParticipantReceivedCard(game, p(1), card10s)

	state = cc.GetVariantState().(*CouponsAndClippingsState)
	a.Equal(int64(1), state.ExpiredPlayerID, "expired player should be 1")
	a.True(state.CouponsExpired, "coupons should be expired")

	// Advance to next round and deal 10♦ face-up to player 3
	game.round++
	card10d := deck.CardFromString("10d")
	card10d.SetBit(faceUp)
	p(3).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(3), card10d)

	state = cc.GetVariantState().(*CouponsAndClippingsState)
	a.Equal(int64(3), state.NailClippingsPlayerID, "nail clippings player should be 3")
	a.True(state.MealComped, "meal comped should be true")

	// Advance to next round and deal a face-up 2 to player 1
	game.round++
	card2c := deck.CardFromString("2c")
	card2c.SetBit(faceUp)
	p(1).hand.AddCard(card2c)
	cc.ParticipantReceivedCard(game, p(1), card2c)

	state = cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.MealCompPlayerIDs, int64(1), "meal comp players should include 1")
}

func TestCouponsAndClippings_SplashResetsBetweenRounds(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Deal a face-up 3 to player 2
	card3c := deck.CardFromString("3c")
	card3c.SetBit(faceUp)
	p(2).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(2), card3c)

	state := cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.CouponPlayerIDs, int64(2))

	// Deal a non-special card to player 1 in the SAME round
	// Splash state should NOT be reset
	card5c := deck.CardFromString("5c")
	p(1).hand.AddCard(card5c)
	cc.ParticipantReceivedCard(game, p(1), card5c)

	state = cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.CouponPlayerIDs, int64(2), "coupon player should persist within same round")

	// Move to next round
	game.round++

	// Deal a non-special card - this triggers the round change reset
	card6c := deck.CardFromString("6c")
	p(1).hand.AddCard(card6c)
	cc.ParticipantReceivedCard(game, p(1), card6c)

	// Splash state should be reset now
	state = cc.GetVariantState().(*CouponsAndClippingsState)
	a.Empty(state.CouponPlayerIDs, "coupon players should be reset after round change")
}

func TestCouponsAndClippings_Start(t *testing.T) {
	cc := &CouponsAndClippings{
		couponsExpired:        true,
		mealComped:            true,
		couponPlayerIDs:       []int64{1, 2},
		mealCompPlayerIDs:     []int64{3, 4},
		expiredPlayerID:       2,
		nailClippingsPlayerID: 3,
		lastDealRound:         round(5),
	}

	cc.Start()

	a := assert.New(t)
	a.False(cc.couponsExpired)
	a.False(cc.mealComped)
	a.Nil(cc.couponPlayerIDs)
	a.Nil(cc.mealCompPlayerIDs)
	a.Equal(int64(0), cc.expiredPlayerID)
	a.Equal(int64(0), cc.nailClippingsPlayerID)
	a.Equal(round(0), cc.lastDealRound)
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

func TestCouponsAndClippings_FaceDownCardsNoSplash(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Deal a face-down 3 to player 2
	card3c := deck.CardFromString("3c")
	// No faceUp bit = face-down
	p(2).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(2), card3c)

	// Should be wild but no splash
	a.True(card3c.IsWild, "3 should still be wild face-down")
	state := cc.GetVariantState().(*CouponsAndClippingsState)
	a.Empty(state.CouponPlayerIDs, "no splash for face-down coupon")
}

func TestCouponsAndClippings_BothEventsCanTrigger(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Give players a 3 and a 2
	card3c := deck.CardFromString("3c")
	p(1).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(1), card3c)

	card2c := deck.CardFromString("2c")
	p(2).hand.AddCard(card2c)
	cc.ParticipantReceivedCard(game, p(2), card2c)

	a.True(card3c.IsWild, "3 should be wild")
	a.False(card2c.IsWild, "2 should NOT be wild")

	// Deal 10♠ face-up (expire coupons)
	card10s := deck.CardFromString("10s")
	card10s.SetBit(faceUp)
	p(1).hand.AddCard(card10s)
	cc.ParticipantReceivedCard(game, p(1), card10s)

	a.False(card3c.IsWild, "3 should NOT be wild after expiration")
	a.False(card2c.IsWild, "2 should still NOT be wild")

	// Deal 10♦ face-up (meal comp)
	card10d := deck.CardFromString("10d")
	card10d.SetBit(faceUp)
	p(2).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(2), card10d)

	a.False(card3c.IsWild, "3 should still NOT be wild")
	a.True(card2c.IsWild, "2 should be wild after meal comp")

	state := cc.GetVariantState().(*CouponsAndClippingsState)
	a.True(state.CouponsExpired)
	a.True(state.MealComped)
}

func TestCouponsAndClippings_MultipleFaceUpThreesInSameRound(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3, 4}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// Deal face-up 3s to multiple players in the same round
	// This tests the bug fix where only the last player's splash would show

	// Player 1 gets face-up 3
	card3c := deck.CardFromString("3c")
	card3c.SetBit(faceUp)
	p(1).hand.AddCard(card3c)
	cc.ParticipantReceivedCard(game, p(1), card3c)

	state := cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.CouponPlayerIDs, int64(1), "player 1 should be in coupon players")
	a.Len(state.CouponPlayerIDs, 1, "should have 1 coupon player")

	// Player 2 gets a non-special card (previously this would reset the splash)
	card5c := deck.CardFromString("5c")
	p(2).hand.AddCard(card5c)
	cc.ParticipantReceivedCard(game, p(2), card5c)

	state = cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.CouponPlayerIDs, int64(1), "player 1 should still be in coupon players")

	// Player 3 gets face-up 3
	card3d := deck.CardFromString("3d")
	card3d.SetBit(faceUp)
	p(3).hand.AddCard(card3d)
	cc.ParticipantReceivedCard(game, p(3), card3d)

	state = cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.CouponPlayerIDs, int64(1), "player 1 should still be in coupon players")
	a.Contains(state.CouponPlayerIDs, int64(3), "player 3 should be in coupon players")
	a.Len(state.CouponPlayerIDs, 2, "should have 2 coupon players")

	// Player 4 gets face-up 3
	card3h := deck.CardFromString("3h")
	card3h.SetBit(faceUp)
	p(4).hand.AddCard(card3h)
	cc.ParticipantReceivedCard(game, p(4), card3h)

	state = cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.CouponPlayerIDs, int64(1), "player 1 should still be in coupon players")
	a.Contains(state.CouponPlayerIDs, int64(3), "player 3 should still be in coupon players")
	a.Contains(state.CouponPlayerIDs, int64(4), "player 4 should be in coupon players")
	a.Len(state.CouponPlayerIDs, 3, "should have 3 coupon players")

	// All cards should be wild
	a.True(card3c.IsWild)
	a.True(card3d.IsWild)
	a.True(card3h.IsWild)
}

func TestCouponsAndClippings_MultipleFaceUpTwosInSameRound(t *testing.T) {
	a := assert.New(t)
	opts := DefaultOptions()
	cc := &CouponsAndClippings{}
	opts.Variant = cc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Drain log channel
	drainLogChannel(game)

	// First activate meal comp with 10♦
	card10d := deck.CardFromString("10d")
	card10d.SetBit(faceUp)
	p(1).hand.AddCard(card10d)
	cc.ParticipantReceivedCard(game, p(1), card10d)

	a.True(cc.mealComped, "meal comp should be active")

	// Move to next round so splash state resets
	game.round++

	// Now deal face-up 2s to multiple players
	card2c := deck.CardFromString("2c")
	card2c.SetBit(faceUp)
	p(1).hand.AddCard(card2c)
	cc.ParticipantReceivedCard(game, p(1), card2c)

	state := cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.MealCompPlayerIDs, int64(1))

	// Player 2 gets face-up 2
	card2d := deck.CardFromString("2d")
	card2d.SetBit(faceUp)
	p(2).hand.AddCard(card2d)
	cc.ParticipantReceivedCard(game, p(2), card2d)

	state = cc.GetVariantState().(*CouponsAndClippingsState)
	a.Contains(state.MealCompPlayerIDs, int64(1))
	a.Contains(state.MealCompPlayerIDs, int64(2))
	a.Len(state.MealCompPlayerIDs, 2, "should have 2 meal comp players")

	// Both should be wild
	a.True(card2c.IsWild)
	a.True(card2d.IsWild)
}
