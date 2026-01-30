package guts

import "mondaynightpoker-server/pkg/deck"

// Participant is an individual in the guts game
type Participant struct {
	PlayerID int64
	balance  int
	hand     []*deck.Card
	traded   int // number of cards traded this round (for animation)
}

// NewParticipant returns a new participant
func NewParticipant(playerID int64) *Participant {
	return &Participant{
		PlayerID: playerID,
		hand:     make([]*deck.Card, 0, 3),
	}
}

// AddCard adds a card to the participant's hand
func (p *Participant) AddCard(card *deck.Card) {
	p.hand = append(p.hand, card)
}

// Hand returns a shallow copy of the participant's hand
func (p *Participant) Hand() []*deck.Card {
	return append([]*deck.Card{}, p.hand...)
}

// ClearHand removes all cards from the participant's hand
func (p *Participant) ClearHand() {
	p.hand = make([]*deck.Card, 0, 3)
}

// hasCard returns true if the card is in the participant's hand
func (p *Participant) hasCard(card *deck.Card) bool {
	for _, c := range p.hand {
		if c.Equal(card) {
			return true
		}
	}
	return false
}

// removeCard removes a card from the participant's hand
func (p *Participant) removeCard(card *deck.Card) {
	newHand := make([]*deck.Card, 0, len(p.hand))
	for _, c := range p.hand {
		if !c.Equal(card) {
			newHand = append(newHand, c)
		}
	}
	p.hand = newHand
}
