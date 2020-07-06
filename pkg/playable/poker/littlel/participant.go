package littlel

import (
	"encoding/json"
	"mondaynightpoker-server/pkg/deck"
)

// Participant represents an individual participant in little L
type Participant struct {
	PlayerID int64
	didFold  bool
	balance  int
	hand     deck.Hand

	// currentBet is how much the player has bet in the current round
	currentBet int
}

// MarshalJSON will encode to JSON
func (p Participant) MarshalJSON() ([]byte, error) {
	return json.Marshal(participantJSON{
		PlayerID:   p.PlayerID,
		DidFold:    p.didFold,
		Balance:    p.balance,
		CurrentBet: p.currentBet,
		Hand:       p.hand,
	})
}

func newParticipant(id int64, ante int) *Participant {
	return &Participant{
		PlayerID: id,
		didFold:  false,
		balance:  -1 * ante,
		hand:     make(deck.Hand, 0),
	}
}
