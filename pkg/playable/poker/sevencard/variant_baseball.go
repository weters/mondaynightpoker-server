package sevencard

import (
	"fmt"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
)

// Baseball is a seven-card variant where 3s and 9s are wild and 4s you get an extra card
type Baseball struct {
	extraCards int
}

// Name returns "Baseball"
func (b *Baseball) Name() string {
	return "Baseball"
}

// Start resets any instance variables
func (b *Baseball) Start() {
	b.extraCards = 0
}

// ParticipantReceivedCard is called after the player receives a card
// Sets wilds
func (b *Baseball) ParticipantReceivedCard(game *Game, p *participant, c *deck.Card) {
	if c.Rank == 3 || c.Rank == 9 {
		c.IsWild = true
	}

	if c.Rank == 4 && c.IsBitSet(faceUp) {
		if len(game.idToParticipant) < 7 || b.extraCards < 3 {
			newCard, err := game.deck.Draw()
			if err != nil {
				panic(fmt.Errorf("could not draw card: %v", err))
			}

			game.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "receives an extra card")
			p.hand.AddCard(newCard)
			b.ParticipantReceivedCard(game, p, newCard)

			b.extraCards++
		}
	}
}
