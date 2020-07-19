package sevencard

import "mondaynightpoker-server/pkg/deck"

// participant is an individual player in seven-card poker
type participant struct {
	PlayerID int64 `json:"playerId"`
	hand     deck.Hand
	didFold  bool

	balance    int
	currentBet int

	didWin bool
}

func newParticipant(playerID int64, ante int) *participant {
	return &participant{
		PlayerID: playerID,
		hand:     make(deck.Hand, 0, 11), // room for 7 cards, plus potential 4 extras (e.g., Baseball)
		balance:  -1 * ante,
	}
}

func (p *participant) resetForNewRound() {
	p.currentBet = 0
}
