package aceydeucey

var config = map[string]interface{}{
	"cardBitFields": map[int]string{
		aceStateUndecided: "undecided",
		aceStateLow:       "low",
		aceStateHigh:      "high",
	},
}

// ParticipantState is the participant's current state
type ParticipantState struct {
	GameState *GameState `json:"gameState"`
	Actions   []Action   `json:"actions"`
}

// GameState is the current state of the game
type GameState struct {
	CurrentTurn  int64                  `json:"currentTurn"`
	Round        *Round                 `json:"round"`
	Participants []*Participant         `json:"participants"`
	Config       map[string]interface{} `json:"config"`
}

func (a *AceyDeucey) getParticipantState(playerID int64) *ParticipantState {
	return &ParticipantState{
		GameState: a.getGameState(),
		Actions:   a.getActionsForParticipant(playerID),
	}
}

func (a *AceyDeucey) getGameState() *GameState {
	var currentTurn int64
	if p := a.getCurrentTurn(); p != nil {
		currentTurn = p.PlayerID
	}

	return &GameState{
		CurrentTurn:  currentTurn,
		Round:        a.currentRound,
		Participants: a.orderedParticipants,
		Config:       config,
	}
}
