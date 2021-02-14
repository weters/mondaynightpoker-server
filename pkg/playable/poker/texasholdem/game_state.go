package texasholdem

import "mondaynightpoker-server/pkg/deck"

// ParticipantState represents the state of an individual participant
type ParticipantState struct {
	Actions     []Action         `json:"actions"`
	Participant *participantJSON `json:"participant"`
	GameState   *GameState       `json:"gameState"`
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
		par := g.participants[id]
		var cards deck.Hand
		var hand string
		if par.reveal && !par.folded {
			cards = par.cards

			if ha := par.getHandAnalyzer(g.community); ha != nil {
				hand = ha.GetHand().String()
			}
		}

		p[i] = &participantJSON{
			PlayerID: par.PlayerID,
			Balance:  par.balance,
			Cards:    cards,
			Folded:   par.folded,
			Bet:      par.bet,
			Hand:     hand,
			Result:   par.result,
			Winnings: par.winnings,
		}
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

	var hand string
	if ha := p.getHandAnalyzer(g.community); ha != nil {
		hand = ha.GetHand().String()
	}

	return &ParticipantState{
		Actions: g.ActionsForParticipant(id),
		Participant: &participantJSON{
			PlayerID: p.PlayerID,
			Balance:  p.balance,
			Cards:    p.cards,
			Folded:   p.folded,
			Bet:      p.bet,
			Hand:     hand,
			Result:   p.result,
			Winnings: p.winnings,
		},
		GameState: g.getGameState(),
	}
}
