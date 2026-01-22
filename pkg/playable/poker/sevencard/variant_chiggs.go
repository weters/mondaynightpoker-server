package sevencard

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
)

// Chiggs is a seven-card variant with mushroom/antidote mechanics
// All 4s are wild. 4♣ is the "Mushroom" that forces neighbors to fold unless they have an antidote (4♠/4♦/4♥)
type Chiggs struct {
	// mushroomActive is true if a mushroom event is in progress
	mushroomActive bool
	// mushroomHolderID is the player who has/flipped the mushroom
	mushroomHolderID int64
	// pendingResponses tracks neighbors who must still respond to the mushroom
	pendingResponses map[int64]bool
	// playersWithFaceDownMushroom tracks players who can flip a face-down mushroom
	playersWithFaceDownMushroom map[int64]*deck.Card
	// lockedMushrooms tracks mushrooms that can no longer be flipped (bet was placed after deal)
	lockedMushrooms map[int64]bool
	// lastAntidotePlayed stores info about the last antidote played for UI display
	lastAntidotePlayed *antidotePlayedInfo
	// lastMushroomFolds stores info about players who folded due to mushroom for UI display
	lastMushroomFolds []*mushroomFoldInfo
	// gameRef stores a reference to the game for neighbor calculations
	gameRef *Game
}

// antidotePlayedInfo stores information about an antidote being played
type antidotePlayedInfo struct {
	PlayerID int64      `json:"playerId"`
	Card     *deck.Card `json:"card"`
}

// mushroomFoldInfo stores information about a player who folded due to mushroom
type mushroomFoldInfo struct {
	PlayerID int64 `json:"playerId"`
}

// ChiggsState is the variant state sent to clients
type ChiggsState struct {
	MushroomActive   bool                `json:"mushroomActive"`
	MushroomHolderID int64               `json:"mushroomHolderId"`
	AntidotePlayed   *antidotePlayedInfo `json:"antidotePlayed,omitempty"`
	MushroomFolds    []*mushroomFoldInfo `json:"mushroomFolds,omitempty"`
	CanFlipMushroom  bool                `json:"canFlipMushroom"`
}

// Name returns "7 Card Chiggs"
func (c *Chiggs) Name() string {
	return "7 Card Chiggs"
}

// Start resets all variant state
func (c *Chiggs) Start() {
	c.mushroomActive = false
	c.mushroomHolderID = 0
	c.pendingResponses = make(map[int64]bool)
	c.playersWithFaceDownMushroom = make(map[int64]*deck.Card)
	c.lockedMushrooms = make(map[int64]bool)
	c.lastAntidotePlayed = nil
	c.lastMushroomFolds = nil
	c.gameRef = nil
}

// ParticipantReceivedCard is called after a participant receives a card
// Sets wilds and triggers mushroom events
func (c *Chiggs) ParticipantReceivedCard(game *Game, p *participant, card *deck.Card) {
	c.gameRef = game

	// All 4s are wild
	if card.Rank == 4 {
		card.IsWild = true

		// Mark as mushroom or antidote
		if card.Suit == deck.Clubs {
			card.SetBit(isMushroom)
		} else {
			card.SetBit(isAntidote)
		}

		// If face-up mushroom, trigger event immediately (not flipped, so card stays in hand)
		if card.IsBitSet(faceUp) && card.IsBitSet(isMushroom) {
			c.triggerMushroomEvent(game, p, card, false)
		} else if !card.IsBitSet(faceUp) && card.IsBitSet(isMushroom) {
			// Track face-down mushroom for optional flip
			c.playersWithFaceDownMushroom[p.PlayerID] = card
		}
	}
}

// triggerMushroomEvent handles a mushroom being revealed (either dealt face-up or flipped)
// wasFlipped indicates if the player chose to flip a face-down mushroom (should discard)
// vs dealt face-up (should keep as wild)
func (c *Chiggs) triggerMushroomEvent(game *Game, p *participant, mushroomCard *deck.Card, wasFlipped bool) {
	c.mushroomActive = true
	c.mushroomHolderID = p.PlayerID
	c.pendingResponses = make(map[int64]bool)
	c.lastAntidotePlayed = nil
	c.lastMushroomFolds = nil

	neighbors := c.getNeighbors(game, p.PlayerID)

	game.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} reveals a mushroom!")

	for _, neighborID := range neighbors {
		neighbor := game.idToParticipant[neighborID]
		if neighbor.didFold {
			continue
		}

		// Check if neighbor has an antidote
		antidoteCard := c.findAntidote(neighbor)
		if antidoteCard != nil {
			// Neighbor has antidote - must use it
			c.pendingResponses[neighborID] = true
		} else {
			// No antidote - track for UI display, then fold
			c.lastMushroomFolds = append(c.lastMushroomFolds, &mushroomFoldInfo{
				PlayerID: neighborID,
			})
			neighbor.didFold = true
			game.logChan <- playable.SimpleLogMessageSlice(neighborID, "{} has no antidote and folds!")
		}
	}

	// Advance decision past any players who just folded
	game.advanceDecisionIfPlayerDidFold()

	// If no pending responses, mushroom phase is complete
	if len(c.pendingResponses) == 0 {
		c.mushroomActive = false
		c.checkForWinByFold(game)
	}

	// Only discard the mushroom if it was voluntarily flipped by the player
	// Face-up dealt mushrooms stay in hand as wilds
	if wasFlipped {
		mushroomCard.SetBit(wasDiscarded)
	}
}

// getNeighbors returns the player IDs of the left and right neighbors with wrap-around
func (c *Chiggs) getNeighbors(game *Game, playerID int64) []int64 {
	var playerIndex int
	for i, id := range game.playerIDs {
		if id == playerID {
			playerIndex = i
			break
		}
	}

	numPlayers := len(game.playerIDs)
	leftIndex := (playerIndex - 1 + numPlayers) % numPlayers
	rightIndex := (playerIndex + 1) % numPlayers

	neighbors := make([]int64, 0, 2)
	if leftIndex != playerIndex {
		neighbors = append(neighbors, game.playerIDs[leftIndex])
	}
	if rightIndex != playerIndex && rightIndex != leftIndex {
		neighbors = append(neighbors, game.playerIDs[rightIndex])
	}

	return neighbors
}

// findAntidote finds an antidote card in the participant's hand
func (c *Chiggs) findAntidote(p *participant) *deck.Card {
	for _, card := range p.hand {
		if card.IsBitSet(isAntidote) && !card.IsBitSet(wasDiscarded) {
			return card
		}
	}
	return nil
}

// GetVariantActions returns additional actions for the player
//
//nolint:revive // participant is intentionally unexported
func (c *Chiggs) GetVariantActions(game *Game, p *participant) []Action {
	actions := make([]Action, 0)

	// If mushroom is active and this player needs to respond
	if c.mushroomActive && c.pendingResponses[p.PlayerID] {
		actions = append(actions, ActionPlayAntidote)
	}

	// If player has a face-down mushroom they can flip (and no mushroom event is active)
	// Cannot flip if:
	// - Mushroom event is active
	// - Mushroom is locked (bet was placed after it was dealt)
	// - Round is over (all players have acted)
	if c.playersWithFaceDownMushroom[p.PlayerID] != nil {
		card := c.playersWithFaceDownMushroom[p.PlayerID]
		roundOver := game.getCurrentTurn() == nil && game.round <= finalBettingRound
		canFlipMushroom := !c.mushroomActive &&
			!c.lockedMushrooms[p.PlayerID] &&
			!roundOver
		if canFlipMushroom && !card.IsBitSet(wasDiscarded) {
			actions = append(actions, ActionFlipMushroom)
		}
	}

	return actions
}

// HandleVariantAction handles variant-specific actions
//
//nolint:revive // participant is intentionally unexported
func (c *Chiggs) HandleVariantAction(game *Game, p *participant, action Action) (bool, error) {
	switch action {
	case ActionFlipMushroom:
		return c.handleFlipMushroom(game, p)
	case ActionPlayAntidote:
		return c.handlePlayAntidote(game, p)
	default:
		return false, nil
	}
}

// handleFlipMushroom handles a player choosing to flip their face-down mushroom
func (c *Chiggs) handleFlipMushroom(game *Game, p *participant) (bool, error) {
	mushroomCard := c.playersWithFaceDownMushroom[p.PlayerID]
	if mushroomCard == nil || mushroomCard.IsBitSet(wasDiscarded) {
		return false, nil
	}

	// Remove from tracking
	delete(c.playersWithFaceDownMushroom, p.PlayerID)

	// Trigger the mushroom event (flipped, so card is discarded)
	c.triggerMushroomEvent(game, p, mushroomCard, true)

	return true, nil
}

// handlePlayAntidote handles a neighbor playing their antidote
func (c *Chiggs) handlePlayAntidote(game *Game, p *participant) (bool, error) {
	if !c.pendingResponses[p.PlayerID] {
		return false, nil
	}

	antidoteCard := c.findAntidote(p)
	if antidoteCard == nil {
		return false, nil
	}

	// Mark antidote as discarded
	antidoteCard.SetBit(wasDiscarded)
	antidoteCard.IsWild = false // No longer wild once discarded

	// Store for UI display
	c.lastAntidotePlayed = &antidotePlayedInfo{
		PlayerID: p.PlayerID,
		Card:     antidoteCard,
	}

	// Remove from pending responses
	delete(c.pendingResponses, p.PlayerID)

	game.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} plays an antidote!")

	// If no more pending responses, mushroom phase is complete
	if len(c.pendingResponses) == 0 {
		c.mushroomActive = false
		c.checkForWinByFold(game)
	}

	return true, nil
}

// checkForWinByFold checks if only one player remains after folding
func (c *Chiggs) checkForWinByFold(game *Game) {
	activePlayers := 0
	for _, p := range game.idToParticipant {
		if !p.didFold {
			activePlayers++
		}
	}

	if activePlayers == 1 {
		game.endGame()
	}
}

// IsVariantPhasePending returns true if waiting for player actions
func (c *Chiggs) IsVariantPhasePending() bool {
	return c.mushroomActive && len(c.pendingResponses) > 0
}

// GetVariantState returns the variant state for clients
func (c *Chiggs) GetVariantState() interface{} {
	return &ChiggsState{
		MushroomActive:   c.mushroomActive,
		MushroomHolderID: c.mushroomHolderID,
		AntidotePlayed:   c.lastAntidotePlayed,
		MushroomFolds:    c.lastMushroomFolds,
		CanFlipMushroom:  false, // This will be set per-player in getPlayerStateByPlayerID
	}
}

// GetVariantStateForPlayer returns variant state customized for a specific player
func (c *Chiggs) GetVariantStateForPlayer(playerID int64) *ChiggsState {
	canFlip := false
	if card := c.playersWithFaceDownMushroom[playerID]; card != nil && !card.IsBitSet(wasDiscarded) {
		// Cannot flip if:
		// - Mushroom event is active
		// - Mushroom is locked (bet was placed after it was dealt)
		// - Round is over (all players have acted)
		roundOver := c.gameRef != nil && c.gameRef.getCurrentTurn() == nil && c.gameRef.round <= finalBettingRound
		canFlip = !c.mushroomActive &&
			!c.lockedMushrooms[playerID] &&
			!roundOver
	}

	return &ChiggsState{
		MushroomActive:   c.mushroomActive,
		MushroomHolderID: c.mushroomHolderID,
		AntidotePlayed:   c.lastAntidotePlayed,
		MushroomFolds:    c.lastMushroomFolds,
		CanFlipMushroom:  canFlip,
	}
}

// ClearAntidotePlayed clears the last antidote played info (called after state is sent)
func (c *Chiggs) ClearAntidotePlayed() {
	c.lastAntidotePlayed = nil
}

// ClearMushroomFolds clears the mushroom folds info (called after state is sent)
func (c *Chiggs) ClearMushroomFolds() {
	c.lastMushroomFolds = nil
}

// OnBetPlaced locks all face-down mushrooms when a bet is placed
func (c *Chiggs) OnBetPlaced(_ *Game) {
	for playerID := range c.playersWithFaceDownMushroom {
		c.lockedMushrooms[playerID] = true
	}
}
