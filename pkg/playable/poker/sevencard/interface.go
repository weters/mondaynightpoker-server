package sevencard

import "mondaynightpoker-server/pkg/deck"

// Variant is a specific variant of seven-card poker (i.e., Stud, Baseball, Chicago, etc.)
type Variant interface {
	// Name should return the name of the game
	Name() string

	// Start resets all variant state
	Start()

	// ParticipantReceivedCard is called after the participant is dealt a new card
	ParticipantReceivedCard(game *Game, p *participant, c *deck.Card)
}

// SplitPotVariant is a variant that splits the pot between the best hand and another condition
type SplitPotVariant interface {
	// GetSplitPotWinners returns the winners for the split portion of the pot,
	// the winning card (if applicable), and a description for the log message.
	// Returns nil if there are no split pot winners.
	GetSplitPotWinners(game *Game) (winners []*participant, card *deck.Card, description string)
}

// InteractiveVariant is a variant that supports custom player actions during the game
type InteractiveVariant interface {
	Variant
	// GetVariantActions returns additional actions available to the player based on variant rules
	GetVariantActions(game *Game, p *participant) []Action
	// HandleVariantAction handles a variant-specific action. Returns true if the action was handled.
	HandleVariantAction(game *Game, p *participant, action Action) (handled bool, err error)
	// IsVariantPhasePending returns true if the variant is waiting for player actions before continuing
	IsVariantPhasePending() bool
	// GetVariantState returns variant-specific state to be included in the game state
	GetVariantState() interface{}
}

// BetAwareVariant is a variant that needs to respond when bets are placed
type BetAwareVariant interface {
	// OnBetPlaced is called when any player places a bet or raise
	OnBetPlaced(game *Game)
}
