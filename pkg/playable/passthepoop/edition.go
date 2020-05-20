package passthepoop

import "mondaynightpoker-server/pkg/deck"

// Edition provides capabilities for a specific variant of Pass the Poop
type Edition interface {
	// Name returns the name of the Edition
	Name() string
	// ParticipantWasPassed  performs any actions on a pass back
	ParticipantWasPassed(participant *Participant, nextCard *deck.Card)


	// TODO
	// We need to figure out how we want to calculate the end of a round
	// We also want to know how we can do it so we can dramatically reveal the results

	// EndRound performs all end of round calculations
	// EndRound(participants []*Participant)
}

