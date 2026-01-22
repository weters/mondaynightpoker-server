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
