package littlel

import "mondaynightpoker-server/pkg/deck"

// Participant represents an individual participant in little L
type Participant struct {
	PlayerID int64 `json:"playerId"`
	didFold  bool
	balance  int
	hand     deck.Hand
}

func newParticipant(id int64, ante int) *Participant {
	return &Participant{
		PlayerID: id,
		didFold:  false,
		balance:  -1 * ante,
		hand:     make(deck.Hand, 0),
	}
}
