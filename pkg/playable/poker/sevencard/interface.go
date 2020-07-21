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
