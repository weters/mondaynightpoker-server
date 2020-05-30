package passthepoop

import (
	"encoding/json"
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
	GameState        *GameState
	Card             *deck.Card
	AvailableActions []GameAction
}

type participantStateJSON struct {
	participantJSON
	GameState        *GameState   `json:"gameState"`
	Card             *deck.Card   `json:"card"`
	AvailableActions []GameAction `json:"availableActions"`
}

// MarshalJSON performs custom JSON marshaling so we don't have to publicly expose
// private fields
func (p *ParticipantState) MarshalJSON() ([]byte, error) {
	var pj participantJSON
	var card *deck.Card
	if participant := p.Participant; participant != nil {
		pj = participant.jsonObject()
		card = participant.card
	}

	return json.Marshal(participantStateJSON{
		participantJSON:  pj,
		GameState:        p.GameState,
		Card:             card,
		AvailableActions: p.AvailableActions,
	})
}
