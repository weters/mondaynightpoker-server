package sevencard

import "mondaynightpoker-server/pkg/deck"

// GameState contains the state about the game
type GameState struct {
	Name         string             `json:"name"`
	Participants []*participantJSON `json:"participants"`
	CurrentTurn  int64              `json:"currentTurn"`
	Round        round              `json:"round"`
	Pot          int                `json:"pot"`
	Ante         int                `json:"ante"`
	CurrentBet   int                `json:"currentBet"`
	MaxBet       int                `json:"maxBet"`
	Winners      []int64            `json:"winners"`
}

func (g *Game) getGameState() GameState {
	var currentTurn int64
	if p := g.getCurrentTurn(); p != nil {
		currentTurn = p.PlayerID
	}

	var winners []int64
	if g.winners != nil {
		winners = make([]int64, len(g.winners))
		for i, winner := range g.winners {
			winners[i] = winner.PlayerID
		}
	}

	isGameOver := g.isGameOver()

	participants := make([]*participantJSON, len(g.playerIDs))
	for i, id := range g.playerIDs {
		p := g.idToParticipant[id]

		var hand deck.Hand
		var handRank string
		if !p.didFold {
			hand = make(deck.Hand, len(p.hand))
			for i, card := range p.hand {
				if isGameOver || card.State&faceUp > 0 {
					hand[i] = card
				} else {
					hand[i] = nil
				}
			}

			if isGameOver && !p.didFold {
				handRank = p.getHandAnalyzer().GetHand().String()
			}
		}

		participants[i] = &participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.balance,
			CurrentBet: p.currentBet,
			Hand:       hand,
			HandRank:   handRank,
		}
	}

	gs := GameState{
		Name:         g.Name(),
		Participants: participants,
		CurrentTurn:  currentTurn,
		Round:        g.round,
		Pot:          g.pot,
		Ante:         g.options.Ante,
		CurrentBet:   g.currentBet,
		MaxBet:       g.getMaxBet(),
		Winners:      winners,
	}

	return gs
}

// PlayerState is the state for the requesting player
type PlayerState struct {
	GameState   GameState        `json:"gameState"`
	Actions     []Action         `json:"actions"`
	Participant *participantJSON `json:"participant"`
}

func (g *Game) getPlayerStateByPlayerID(playerID int64) PlayerState {
	var pJSON *participantJSON
	var actions []Action
	if p, ok := g.idToParticipant[playerID]; ok {
		pJSON = &participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.balance,
			CurrentBet: p.currentBet,
			Hand:       p.hand,
			HandRank:   p.getHandAnalyzer().GetHand().String(),
		}

		actions = g.getActionsForParticipant(p)
	}

	return PlayerState{
		GameState:   g.getGameState(),
		Actions:     actions,
		Participant: pJSON,
	}
}

type participantJSON struct {
	PlayerID   int64     `json:"playerId"`
	DidFold    bool      `json:"didFold"`
	Balance    int       `json:"balance"`
	CurrentBet int       `json:"currentBet"`
	Hand       deck.Hand `json:"hand"`
	HandRank   string    `json:"handRank"`
}
