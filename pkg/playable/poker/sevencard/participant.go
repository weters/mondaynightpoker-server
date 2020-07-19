package sevencard

import "mondaynightpoker-server/pkg/deck"

// participant is an individual player in seven-card poker
type participant struct {
	PlayerID int64 `json:"playerId"`
	hand     deck.Hand
	didFold  bool
}

func newParticipant(playerID int64) *participant {
	return &participant{
		PlayerID: playerID,
		hand:     make(deck.Hand, 0, 11), // room for 7 cards, plus potential 4 extras (e.g., Baseball)
	}
}
