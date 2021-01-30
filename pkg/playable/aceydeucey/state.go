package aceydeucey

// ParticipantState is the participant's current state
type ParticipantState struct {
	GameState *GameState `json:"gameState"`
}

// GameState is the current state of the game
type GameState struct {
}
