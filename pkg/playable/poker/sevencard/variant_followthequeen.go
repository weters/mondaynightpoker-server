package sevencard

import (
	"mondaynightpoker-server/pkg/deck"
)

// FollowTheQueen is follow the queen
type FollowTheQueen struct {
	wildRank        int
	queenWasFlipped bool
}

// Name returns the name
func (f *FollowTheQueen) Name() string {
	return "Follow the Queen"
}

// Start starts the game
func (f *FollowTheQueen) Start() {
	f.wildRank = 0
	f.queenWasFlipped = false
}

// ParticipantReceivedCard will update the wilds
func (f *FollowTheQueen) ParticipantReceivedCard(game *Game, p *participant, c *deck.Card) {
	wildDidChange := false
	if c.IsBitSet(faceUp) && c.Rank == deck.Queen {
		f.queenWasFlipped = true
		wildDidChange = true
	} else {
		if f.queenWasFlipped && c.IsBitSet(faceUp) {
			f.wildRank = c.Rank
			wildDidChange = true
		}

		f.queenWasFlipped = false
	}

	if wildDidChange {
		for _, participant := range game.idToParticipant {
			for _, card := range participant.hand {
				card.IsWild = card.Rank == f.wildRank || card.Rank == deck.Queen
			}
		}
	} else {
		c.IsWild = c.Rank == f.wildRank || c.Rank == deck.Queen
	}
}
