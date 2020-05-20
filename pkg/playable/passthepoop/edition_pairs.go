package passthepoop

import "mondaynightpoker-server/pkg/deck"

// PairsEdition is a variant of Pass the Poop where pairs on the board good
// Any pair on the board is better than any single card
// Trips or better on the board and the rest of the board loses all their lives
type PairsEdition struct {

}

// Name returns the name of the Edition
func (p *PairsEdition) Name() string {
	return "Pairs"
}

// ParticipantWasPassed is a no-op in pairs edition
func (p *PairsEdition) ParticipantWasPassed(participant *Participant, nextCard *deck.Card) {
	// noop
}
