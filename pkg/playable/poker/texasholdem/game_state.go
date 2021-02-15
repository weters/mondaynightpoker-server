package texasholdem

import "mondaynightpoker-server/pkg/deck"

// ParticipantState represents the state of an individual participant
type ParticipantState struct {
	Actions       []Action         `json:"actions"`
	FutureActions []Action         `json:"futureActions"`
	Participant   *participantJSON `json:"participant"`
	GameState     *GameState       `json:"gameState"`
}

// GameState represents the state of the game
type GameState struct {
	Name         string             `json:"name"`
	DealerState  DealerState        `json:"dealerState"`
	Community    deck.Hand          `json:"community"`
	Pot          int                `json:"pot"`
	CurrentBet   int                `json:"currentBet"`
	Participants []*participantJSON `json:"participants"`
	CurrentTurn  int64              `json:"currentTurn"`
	LastAction   *lastAction        `json:"lastAction"`
}

func (g *Game) getGameState() *GameState {
	p := make([]*participantJSON, len(g.participants))
	for i, id := range g.participantOrder {
		p[i] = g.participants[id].participantJSON(g, false)
	}

	var currentTurn int64 = 0
	if turn, _ := g.GetCurrentTurn(); turn != nil {
		currentTurn = turn.PlayerID
	}

	return &GameState{
		Name:         g.Name(),
		DealerState:  g.dealerState,
		Pot:          g.pot,
		CurrentBet:   g.currentBet,
		Participants: p,
		CurrentTurn:  currentTurn,
		Community:    g.community,
		LastAction:   g.lastAction,
	}
}

func (g *Game) getParticipantStateByPlayerID(id int64) *ParticipantState {
	p := g.participants[id]

	return &ParticipantState{
		Actions:       g.ActionsForParticipant(id),
		FutureActions: g.FutureActionsForParticipant(id),
		Participant:   p.participantJSON(g, true),
		GameState:     g.getGameState(),
	}
}
