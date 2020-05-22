package passthepoop

import "mondaynightpoker-server/pkg/deck"

// GameState is the overall state of the game
// These values must be safe for someone to snoop on
type GameState struct {
	Edition         string                 `json:"edition"`
	Participants    []*Participant         `json:"participants"`
	AllParticipants map[int64]*Participant `json:"allParticipants"`
	Ante            int                    `json:"ante"`
	Pot             int                    `json:"pot"`
	DecisionIndex   int                    `json:"decisionIndex"`
}

// ParticipantState is the state for a specific participant
type ParticipantState struct {
	*Participant
	GameState *GameState `json:"gameState"`
	Card      *deck.Card `json:"card"`
}
