package texasholdem

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
)

// ParticipantState represents the state of an individual participant
type ParticipantState struct {
	Actions       []action.Action  `json:"actions"`
	FutureActions []action.Action  `json:"futureActions"`
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
	for i, pt := range g.participantOrder {
		p[i] = g.participants[pt.ID()].participantJSON(g, false)
	}

	var currentTurn int64 = 0
	if turn, _ := g.GetCurrentTurn(); turn != nil {
		currentTurn = turn.PlayerID
	}

	return &GameState{
		Name:         g.Name(),
		DealerState:  g.dealerState,
		Pot:          0, // FIXME
		CurrentBet:   g.potManager.GetBet(),
		Participants: p,
		CurrentTurn:  currentTurn,
		Community:    g.community,
		LastAction:   g.lastAction,
	}
}

func (g *Game) getParticipantStateByPlayerID(id int64) *ParticipantState {
	var pjson *participantJSON
	var actions, futureActions []action.Action
	if p, ok := g.participants[id]; ok {
		// force reveal because it's for the current player
		pjson = p.participantJSON(g, true)
		actions = g.ActionsForParticipant(id)
		futureActions = g.FutureActionsForParticipant(id)
	}

	return &ParticipantState{
		Actions:       actions,
		FutureActions: futureActions,
		Participant:   pjson,
		GameState:     g.getGameState(),
	}
}
