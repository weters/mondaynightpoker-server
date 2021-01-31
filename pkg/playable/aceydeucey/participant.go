package aceydeucey

// Participant represents an active particpant in the game
type Participant struct {
	PlayerID int64 `json:"playerId"`
	Balance  int   `json:"balance"`
}

// NewParticipant returns a new participant
func NewParticipant(playerID int64, ante int) *Participant {
	return &Participant{
		PlayerID: playerID,
		Balance:  -1 * ante,
	}
}
