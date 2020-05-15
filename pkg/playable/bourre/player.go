package bourre

import (
	"mondaynightpoker-server/pkg/deck"
)

// Player is an individual in the game
type Player struct {
	PlayerID int64
	balance  int
	hand     []*deck.Card
	folded   bool
	winCount int
}

// NewPlayer returns a new player
func NewPlayer(pid int64) *Player {
	return &Player{
		PlayerID: pid,
		hand:     make([]*deck.Card, 0),
	}
}

// Fold will fold the players hand
func (p *Player) Fold() {
	p.folded = true
}

// AddCard add a card to the players hand
func (p *Player) AddCard(card *deck.Card) {
	p.hand = append(p.hand, card)
}

// Hand returns a shallow clone of the player's hand
func (p *Player) Hand() []*deck.Card {
	return append([]*deck.Card{}, p.hand...)
}

// HasCard returns true if the player has the card in their hand
func (p *Player) HasCard(card *deck.Card) bool {
	for _, c := range p.hand {
		if card.Equal(c) {
			return true
		}
	}

	return false
}

// playerDidPlayCard removes the card from the player's hand
func (p *Player) playerDidPlayCard(card *deck.Card) error {
	hand := make([]*deck.Card, 0)
	found := false
	for _, c := range p.hand {
		if c.Equal(card) {
			found = true
		} else {
			hand = append(hand, c)
		}
	}

	if !found {
		return ErrCardNotInPlayersHand
	}

	p.hand = hand

	return nil
}

// NewRound is called after a new round starts to reset values
func (p *Player) NewRound() {
}

// WonRound marks the player as winning a round
func (p *Player) WonRound() {
	p.winCount++
}

// NewGame is called for a new game
func (p *Player) NewGame() {
	p.winCount = 0
	p.folded = false
	p.hand = make([]*deck.Card, 0)
}
