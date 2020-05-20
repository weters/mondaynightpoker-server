package passthepoop

import "mondaynightpoker-server/pkg/deck"

// DiarrheaEdition is a faster variant of Pass the Poop with the following stipulations
// 1. An ace passed back results in the loss of a life, the ace is thrown out, and we still play for next low card
// 2. If multiple people are tied with the lowest card, they lose all their lives, the cards are thrown out, and we still play for next low card
type DiarrheaEdition struct {

}

// Name returns the name of the Edition
func (d *DiarrheaEdition) Name() string {
	return "Diarrhea"
}

// ParticipantWasPassed will check if an ace was passed back
func (d *DiarrheaEdition) ParticipantWasPassed(participant *Participant, nextCard *deck.Card) {
	if nextCard.Rank == deck.Ace {
		participant.deadCard = true
		participant.subtractLife(1)
	}
}
