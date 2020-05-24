package passthepoop

import (
	"encoding/json"
	"log"
	"mondaynightpoker-server/pkg/deck"
)

// GameState is the overall state of the game
// These values must be safe for someone to snoop on
type GameState struct {
	Edition         string                 `json:"edition"`
	Participants    []*Participant         `json:"participants"`
	AllParticipants map[int64]*Participant `json:"allParticipants"`
	Ante            int                    `json:"ante"`
	Pot             int                    `json:"pot"`
	CurrentTurn     int64                  `json:"currentTurn"`
}

// ParticipantState is the state for a specific participant
type ParticipantState struct {
	*Participant
	GameState *GameState `json:"gameState"`
	Card      *deck.Card `json:"card"`
}

type participantStateJSON struct {
	participantJSON
	GameState *GameState `json:"gameState"`
	Card      *deck.Card `json:"card"`
}

// MarshalJSON performs custom JSON marshaling so we don't have to publicly expose
// private fields
func (p *ParticipantState) MarshalJSON() ([]byte, error) {
	log.Println("MJ")
	return json.Marshal(participantStateJSON{
		participantJSON: p.Participant.jsonObject(),
		GameState:       p.GameState,
		Card:            p.Participant.card,
	})
}
