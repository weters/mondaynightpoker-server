package sevencard

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
)

// CouponsAndClippings is a seven-card variant themed around couponing and nail clippings
// 3s are wild (coupons) unless 10♠ is dealt face-up (expired)
// 2s become wild (meal comp'd) after 10♦ is dealt face-up (nail clippings)
type CouponsAndClippings struct {
	// couponsExpired is true after 10♠ is dealt face-up
	couponsExpired bool
	// mealComped is true after 10♦ is dealt face-up
	mealComped bool

	// Track ALL players who triggered splash events in current deal round
	// Arrays are used because multiple players can receive splash-worthy cards
	// (e.g., multiple 3s can be dealt face-up in the same round)
	couponPlayerIDs       []int64 // Players who got face-up 3s (before expiration)
	mealCompPlayerIDs     []int64 // Players who got face-up 2s (after meal comp)
	expiredPlayerID       int64   // Only one 10♠ can trigger this
	nailClippingsPlayerID int64   // Only one 10♦ can trigger this

	// Track which round we last dealt cards in (to know when to reset arrays)
	lastDealRound round
}

// CouponsAndClippingsState is the variant state sent to clients
type CouponsAndClippingsState struct {
	CouponsExpired        bool    `json:"couponsExpired"`
	MealComped            bool    `json:"mealComped"`
	CouponPlayerIDs       []int64 `json:"couponPlayerIds,omitempty"`
	MealCompPlayerIDs     []int64 `json:"mealCompPlayerIds,omitempty"`
	ExpiredPlayerID       int64   `json:"expiredPlayerId,omitempty"`
	NailClippingsPlayerID int64   `json:"nailClippingsPlayerId,omitempty"`
}

// Name returns "Coupons and Clippings"
func (c *CouponsAndClippings) Name() string {
	return "Coupons and Clippings"
}

// Start resets all variant state
func (c *CouponsAndClippings) Start() {
	c.couponsExpired = false
	c.mealComped = false
	c.couponPlayerIDs = nil
	c.mealCompPlayerIDs = nil
	c.expiredPlayerID = 0
	c.nailClippingsPlayerID = 0
	c.lastDealRound = 0
}

// ParticipantReceivedCard is called after a participant receives a card
// Sets wilds and triggers special events
func (c *CouponsAndClippings) ParticipantReceivedCard(game *Game, p *participant, card *deck.Card) {
	// Reset splash state when entering a new deal round
	// This ensures all players who receive splash-worthy cards in the same round
	// are tracked (fixing the bug where only the last player's splash showed)
	if game.round != c.lastDealRound {
		c.lastDealRound = game.round
		c.couponPlayerIDs = nil
		c.mealCompPlayerIDs = nil
		c.expiredPlayerID = 0
		c.nailClippingsPlayerID = 0
	}

	// 10♠ face-up = Coupons Expired
	if card.Rank == 10 && card.Suit == deck.Spades && card.IsBitSet(faceUp) {
		c.expireCoupons(game)
		c.expiredPlayerID = p.PlayerID
		game.logChan <- playable.SimpleLogMessageSlice(0, "The 10 of Spades was dealt! All coupons (3s) have expired!")
		return
	}

	// 10♦ face-up = Nail Clippings (meal comp)
	if card.Rank == 10 && card.Suit == deck.Diamonds && card.IsBitSet(faceUp) {
		c.activateMealComp(game)
		c.nailClippingsPlayerID = p.PlayerID
		game.logChan <- playable.SimpleLogMessageSlice(0, "Nail clippings found! All 2s are now wild!")
		return
	}

	// 3s are wild (if not expired)
	if card.Rank == 3 {
		if !c.couponsExpired {
			card.IsWild = true
			if card.IsBitSet(faceUp) {
				c.couponPlayerIDs = append(c.couponPlayerIDs, p.PlayerID)
			}
		}
		return
	}

	// 2s are wild (if meal comped)
	if card.Rank == 2 {
		if c.mealComped {
			card.IsWild = true
			if card.IsBitSet(faceUp) {
				c.mealCompPlayerIDs = append(c.mealCompPlayerIDs, p.PlayerID)
			}
		}
		return
	}
}

// expireCoupons removes wild status from all 3s
func (c *CouponsAndClippings) expireCoupons(game *Game) {
	c.couponsExpired = true

	// Remove wild status from all 3s in all hands
	for _, participant := range game.idToParticipant {
		for _, card := range participant.hand {
			if card.Rank == 3 {
				card.IsWild = false
			}
		}
	}
}

// activateMealComp makes all 2s wild
func (c *CouponsAndClippings) activateMealComp(game *Game) {
	c.mealComped = true

	// Make all 2s wild in all hands
	for _, participant := range game.idToParticipant {
		for _, card := range participant.hand {
			if card.Rank == 2 {
				card.IsWild = true
			}
		}
	}
}

// GetVariantActions returns additional actions for the player (CouponsAndClippings has none)
//
//nolint:revive // participant is intentionally unexported
func (c *CouponsAndClippings) GetVariantActions(_ *Game, _ *participant) []Action {
	return nil
}

// HandleVariantAction handles variant-specific actions (CouponsAndClippings has none)
//
//nolint:revive // participant is intentionally unexported
func (c *CouponsAndClippings) HandleVariantAction(_ *Game, _ *participant, _ Action) (bool, error) {
	return false, nil
}

// IsVariantPhasePending returns true if the variant is waiting for player actions (CouponsAndClippings never waits)
func (c *CouponsAndClippings) IsVariantPhasePending() bool {
	return false
}

// GetVariantState returns the variant state for clients
func (c *CouponsAndClippings) GetVariantState() interface{} {
	return &CouponsAndClippingsState{
		CouponsExpired:        c.couponsExpired,
		MealComped:            c.mealComped,
		CouponPlayerIDs:       c.couponPlayerIDs,
		MealCompPlayerIDs:     c.mealCompPlayerIDs,
		ExpiredPlayerID:       c.expiredPlayerID,
		NailClippingsPlayerID: c.nailClippingsPlayerID,
	}
}
