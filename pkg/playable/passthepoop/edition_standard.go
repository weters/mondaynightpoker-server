package passthepoop

import "mondaynightpoker-server/pkg/deck"

// StandardEdition is the standard variant of Pass the Poop
type StandardEdition struct {
}

// Name returns the name of the Edition
func (s *StandardEdition) Name() string {
	return "Standard"
}

// ParticipantWasPassed is a no-op in standard edition
func (s *StandardEdition) ParticipantWasPassed(participant *Participant, nextCard *deck.Card) {
	// noop
}
