package sevencard

import (
	"mondaynightpoker-server/pkg/deck"
)

// Stud is a standard game of seven-card stud
// Two face-down, four face-up, and a final face-down card with
// betting rounds after the third, fourth, fifth, sixth, and final card
type Stud struct {
}

// ParticipantReceivedCard is a no-op for stud
func (s *Stud) ParticipantReceivedCard(_ *Game, _ *participant, _ *deck.Card) {
}

// Name returns "Seven-Card Stud"
func (s *Stud) Name() string {
	return "Seven-Card Stud"
}

// Start is a no-op
func (s *Stud) Start() {

}
