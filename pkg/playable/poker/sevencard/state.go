package sevencard

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker"
)

// GameState contains the state about the game
type GameState struct {
	Name         string             `json:"name"`
	Participants []*participantJSON `json:"participants"`
	CurrentTurn  int64              `json:"currentTurn"`
	DealerID     int64              `json:"dealerId"`
	Round        round              `json:"round"`
	Pot          int                `json:"pot"`
	Ante         int                `json:"ante"`
	CurrentBet   int                `json:"currentBet"`
	MaxBet       int                `json:"maxBet"`
	LastAction   *lastAction        `json:"lastAction"`
	Winners      map[int64]int      `json:"winners"`
}

func (g *Game) getGameState() GameState {
	var currentTurn int64
	if p := g.getCurrentTurn(); p != nil {
		currentTurn = p.PlayerID
	}

	var winners map[int64]int
	if g.winners != nil {
		winners = make(map[int64]int)
		for winner, amount := range g.winners {
			winners[winner.PlayerID] = amount
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
				if isGameOver || card.IsBitSet(faceUp) {
					hand[i] = card.Clone()

					// if it's a private wild (i.e., based on a low-card in the hole),
					// then we don't want to show it to all players
					if !isGameOver && card.IsBitSet(privateWild) {
						hand[i].IsWild = false
					}
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
			Balance:    p.Balance(),
			CurrentBet: p.currentBet,
			Hand:       hand,
			HandRank:   handRank,
		}
	}

	gs := GameState{
		Name:         g.Name(),
		Participants: participants,
		CurrentTurn:  currentTurn,
		DealerID:     g.GetDealerID(),
		Round:        g.round,
		Pot:          g.pot,
		Ante:         g.options.Ante,
		CurrentBet:   g.currentBet,
		MaxBet:       g.getMaxBet(),
		LastAction:   g.lastAction,
		Winners:      winners,
	}

	return gs
}

// PlayerState is the state for the requesting player
type PlayerState struct {
	GameState     GameState        `json:"gameState"`
	Actions       []Action         `json:"actions"`
	FutureActions []Action         `json:"futureActions"`
	Participant   *participantJSON `json:"participant"`
	PokerState    *poker.State     `json:"pokerState"`
}

func (g *Game) getPlayerStateByPlayerID(playerID int64) PlayerState {
	var pJSON *participantJSON
	var actions []Action
	var futureActions []Action
	if p, ok := g.idToParticipant[playerID]; ok {
		pJSON = &participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.Balance(),
			CurrentBet: p.currentBet,
			Hand:       p.hand,
			HandRank:   p.getHandAnalyzer().GetHand().String(),
		}

		actions = g.getActionsForParticipant(p)
		futureActions = g.getFutureActionsForParticipant(p)
	}

	return PlayerState{
		GameState:     g.getGameState(),
		Actions:       actions,
		FutureActions: futureActions,
		Participant:   pJSON,
		PokerState:    g.getPokerState(),
	}
}

func (g *Game) getPokerState() *poker.State {
	minBet := g.options.Ante
	if g.currentBet > 0 {
		minBet = g.currentBet * 2
	}

	return &poker.State{
		Ante:       g.options.Ante,
		CurrentBet: g.currentBet,
		MinBet:     minBet,
		MaxBet:     g.getMaxBet(),
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
