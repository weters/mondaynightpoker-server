package passthepoop

import (
	"encoding/json"
	"mondaynightpoker-server/pkg/deck"
)

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
// Returns the number of lives lost
func (p *Participant) subtractLife(count int) int {
	if count < 0 {
		panic("count cannot be less than 0")
	}

	if count == 0 {
		livesLost := p.lives
		p.lives = 0
		return livesLost
	}

	originalLives := p.lives
	p.lives -= count
	if p.lives < 0 {
		p.lives = 0
	}

	return originalLives - p.lives
}

// -- MarshallJSON implementation --

type participantJSON struct {
	PlayerID int64 `json:"playerId"`
	Balance int `json:"balance"`
	Lives int `json:"lives"`
	IsFlipped bool `json:"isFlipped"`
}

func (p *Participant) jsonObject() participantJSON {
	return participantJSON{
		PlayerID:  p.PlayerID,
		Balance:   p.balance,
		Lives:     p.lives,
		IsFlipped: p.isFlipped,
	}
}

// MarshalJSON will JSON encode the data
// Using a custom marshaller so we can expose some private fields that
// I do not want to make public
func (p *Participant) MarshalJSON() ([]byte, error)  {
	return json.Marshal(p.jsonObject())
}
