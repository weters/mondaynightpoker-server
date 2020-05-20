package passthepoop

import "mondaynightpoker-server/pkg/deck"

// Participant is an individual participant in the game
type Participant struct {
	// PlayerID is the ID of the player in the database
	PlayerID int64 `json:"playerId"`

	// when lives hit 0, player is out of the game
	lives int

	// how much the player is up or down
	balance int

	// the current card the player was dealt
	card *deck.Card

	// if true, this card is not part of the end round calcuation
	deadCard bool

	// whether the card should be shown to the table
	isFlipped bool
}

// newRound is called to reset the participant prior to a new round
func (p *Participant) newRound() {
	p.card = nil
	p.isFlipped = false
}

// subtractLife will subtract the specified number of lives
// if count == 0, subtract all the lives!
func (p *Participant) subtractLife(count int) {
	if count < 0 {
		panic("count cannot be less than 0")
	}

	if count == 0 {
		p.lives = 0
		return
	}

	p.lives -= count
	if p.lives < 0 {
		p.lives = 0
	}
}
