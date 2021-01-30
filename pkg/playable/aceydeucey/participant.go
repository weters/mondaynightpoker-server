package aceydeucey

// Participant represents an active particpant in the game
type Participant struct {
	playerID int64
	balance  int
}

// NewParticipant returns a new participant
func NewParticipant(playerID int64, ante int) *Participant {
	return &Participant{
		playerID: playerID,
		balance:  -1 * ante,
	}
}
