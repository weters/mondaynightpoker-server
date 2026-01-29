package sevencard

import (
	"fmt"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
)

// CouponsAndClippings is a seven-card variant with BOGO wild mechanics and nail clipping refunds
// - No wilds to start
// - BOGO: When a second face-up card of the same rank appears, ALL cards of that rank become wild
// - New BOGO triggers remove wild status from the previous wild rank (only one wild rank at a time)
// - Nail Clipping: Any face-up 10 refunds the player's ante from the pot
type CouponsAndClippings struct {
	// faceUpRankCounts tracks how many face-up cards of each rank have been dealt
	faceUpRankCounts map[int]int
	// currentWildRank is the rank that is currently wild (0 if none)
	currentWildRank int
	// bogoPlayerID is the player who triggered the latest BOGO
	bogoPlayerID int64
	// nailClippingPlayerIDs are players who got ante refunds this round
	nailClippingPlayerIDs []int64
	// lastDealRound tracks which round we last dealt cards in (to know when to reset splash arrays)
	lastDealRound round
}

// CouponsAndClippingsState is the variant state sent to clients
type CouponsAndClippingsState struct {
	CurrentWildRank       int     `json:"currentWildRank"`
	BogoPlayerID          int64   `json:"bogoPlayerId,omitempty"`
	NailClippingPlayerIDs []int64 `json:"nailClippingPlayerIds,omitempty"`
}

// Name returns "Coupons and Clippings"
func (c *CouponsAndClippings) Name() string {
	return "Coupons and Clippings"
}

// Start resets all variant state
func (c *CouponsAndClippings) Start() {
	c.faceUpRankCounts = make(map[int]int)
	c.currentWildRank = 0
	c.bogoPlayerID = 0
	c.nailClippingPlayerIDs = nil
	c.lastDealRound = 0
}

// ParticipantReceivedCard is called after a participant receives a card
// Handles BOGO wild triggers and nail clipping ante refunds
func (c *CouponsAndClippings) ParticipantReceivedCard(game *Game, p *participant, card *deck.Card) {
	// Reset splash state when entering a new deal round
	if game.round != c.lastDealRound {
		c.lastDealRound = game.round
		c.bogoPlayerID = 0
		c.nailClippingPlayerIDs = nil
	}

	// Handle face-down cards: just check if they match current wild rank
	if !card.IsBitSet(faceUp) {
		if c.currentWildRank > 0 && card.Rank == c.currentWildRank {
			card.IsWild = true
		}
		return
	}

	// Face-up card processing

	// Handle nail clipping (face-up 10 refunds ante)
	if card.Rank == 10 {
		c.handleNailClipping(game, p)
	}

	// Increment face-up rank count
	c.faceUpRankCounts[card.Rank]++
	count := c.faceUpRankCounts[card.Rank]

	// Check for BOGO trigger (second face-up card of same rank)
	if count == 2 {
		c.triggerBogo(game, card.Rank, p)
		return
	}

	// If card matches current wild rank, make it wild
	if c.currentWildRank > 0 && card.Rank == c.currentWildRank {
		card.IsWild = true
	}
}

// handleNailClipping refunds the player's ante from the pot if sufficient balance
func (c *CouponsAndClippings) handleNailClipping(game *Game, p *participant) {
	ante := game.options.Ante
	if game.pot >= ante {
		game.pot -= ante
		p.balance += ante
		c.nailClippingPlayerIDs = append(c.nailClippingPlayerIDs, p.PlayerID)
		game.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} found a nail clipping! Ante refunded.")
	} else {
		game.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} found a nail clipping but the pot is too small for a refund.")
	}
}

// triggerBogo triggers a BOGO event, making all cards of the given rank wild
// and removing wild status from the previous wild rank
func (c *CouponsAndClippings) triggerBogo(game *Game, rank int, p *participant) {
	previousWildRank := c.currentWildRank
	c.currentWildRank = rank
	c.bogoPlayerID = p.PlayerID

	// Iterate all players' hands and update wild status
	for _, participant := range game.idToParticipant {
		for _, card := range participant.hand {
			// Remove wild from previous rank
			if previousWildRank > 0 && card.Rank == previousWildRank {
				card.IsWild = false
			}
			// Add wild to new rank
			if card.Rank == rank {
				card.IsWild = true
			}
		}
	}

	game.logChan <- playable.SimpleLogMessageSlice(0, "BOGO! All %ss are now wild!", rankName(rank))
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
		CurrentWildRank:       c.currentWildRank,
		BogoPlayerID:          c.bogoPlayerID,
		NailClippingPlayerIDs: c.nailClippingPlayerIDs,
	}
}

// rankName converts a rank integer to a display name
func rankName(rank int) string {
	switch rank {
	case deck.Jack:
		return "Jack"
	case deck.Queen:
		return "Queen"
	case deck.King:
		return "King"
	case deck.Ace:
		return "Ace"
	default:
		return fmt.Sprintf("%d", rank)
	}
}
