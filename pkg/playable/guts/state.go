package guts

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
)

// GameState is the overall game state
// This is safe for all players to see
type GameState struct {
	Participants []*GameStateParticipant `json:"participants"`
	Pot          int                     `json:"pot"`
	Round        int                     `json:"round"`
	Phase        string                  `json:"phase"`
	MaxOwed      int                     `json:"maxOwed"`
	Ante         int                     `json:"ante"`
	CardCount    int                     `json:"cardCount"`
	IsGameOver   bool                    `json:"isGameOver"`
	// Decisions is only populated during/after showdown
	Decisions map[int64]bool `json:"decisions,omitempty"`
	// ShowdownResult is only populated after showdown
	ShowdownResult *ShowdownResultState `json:"showdownResult,omitempty"`
}

// GameStateParticipant is the state of an individual participant
type GameStateParticipant struct {
	PlayerID int64 `json:"playerId"`
	Balance  int   `json:"balance"`
	// Decided is true if the player has made their in/out decision
	Decided bool `json:"decided"`
	// CardsInHand is the number of cards the player has
	CardsInHand int `json:"cardsInHand"`
	// Hand is only shown after showdown for players who went in
	Hand []*deck.Card `json:"hand,omitempty"`
}

// ShowdownResultState is the showdown result for the state
type ShowdownResultState struct {
	WinnerIDs    []int64 `json:"winnerIds"`
	LoserIDs     []int64 `json:"loserIds"`
	PlayersInIDs []int64 `json:"playersInIds"`
	PotWon       int     `json:"potWon"`
	PenaltyPaid  int     `json:"penaltyPaid"`
	NextPot      int     `json:"nextPot"`
	AllFolded    bool    `json:"allFolded"`
}

// Response is the response format for this game
type Response struct {
	GameState *GameState   `json:"gameState"`
	Balance   int          `json:"balance"`
	Hand      []*deck.Card `json:"hand"`
	// CanDecide is true if the player can make a decision
	CanDecide bool `json:"canDecide"`
	// HasDecided is true if the player has already decided
	HasDecided bool `json:"hasDecided"`
	// MyDecision is the player's decision (only shown after all have decided)
	MyDecision *bool `json:"myDecision,omitempty"`
}

func (g *Game) phaseName() string {
	switch g.phase {
	case PhaseDealing:
		return "dealing"
	case PhaseDeclaration:
		return "declaration"
	case PhaseShowdown:
		return "showdown"
	case PhaseRoundEnd:
		return "roundEnd"
	case PhaseGameOver:
		return "gameOver"
	default:
		return "unknown"
	}
}

func (g *Game) getGameState() *GameState {
	participants := make([]*GameStateParticipant, len(g.participants))
	allDecided := len(g.pendingDecisions) == 0

	for i, p := range g.participants {
		gsp := &GameStateParticipant{
			PlayerID:    p.PlayerID,
			Balance:     p.balance,
			Decided:     !g.pendingDecisions[p.PlayerID],
			CardsInHand: len(p.hand),
		}

		// Show hands of players who went in after showdown
		if allDecided && g.decisions[p.PlayerID] {
			gsp.Hand = p.Hand()
		}

		participants[i] = gsp
	}

	state := &GameState{
		Participants: participants,
		Pot:          g.pot,
		Round:        g.roundNumber,
		Phase:        g.phaseName(),
		MaxOwed:      g.options.MaxOwed,
		Ante:         g.options.Ante,
		CardCount:    g.options.CardCount,
		IsGameOver:   g.phase == PhaseGameOver,
	}

	// Only show decisions after all have decided
	if allDecided && len(g.decisions) > 0 {
		state.Decisions = g.decisions
	}

	// Include showdown result if available
	if g.showdownResult != nil {
		state.ShowdownResult = &ShowdownResultState{
			PotWon:      g.showdownResult.PotWon,
			PenaltyPaid: g.showdownResult.PenaltyPaid,
			NextPot:     g.showdownResult.NextPot,
			AllFolded:   g.showdownResult.AllFolded,
		}

		winnerIDs := make([]int64, len(g.showdownResult.Winners))
		for i, w := range g.showdownResult.Winners {
			winnerIDs[i] = w.PlayerID
		}
		state.ShowdownResult.WinnerIDs = winnerIDs

		loserIDs := make([]int64, len(g.showdownResult.Losers))
		for i, l := range g.showdownResult.Losers {
			loserIDs[i] = l.PlayerID
		}
		state.ShowdownResult.LoserIDs = loserIDs

		playersInIDs := make([]int64, len(g.showdownResult.PlayersIn))
		for i, p := range g.showdownResult.PlayersIn {
			playersInIDs[i] = p.PlayerID
		}
		state.ShowdownResult.PlayersInIDs = playersInIDs
	}

	return state
}

// GetPlayerState returns the state for the given player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	participant, ok := g.idToParticipant[playerID]
	if !ok {
		// Viewer who is not playing
		participant = &Participant{
			PlayerID: playerID,
			balance:  0,
			hand:     nil,
		}
	}

	gameState := g.getGameState()
	allDecided := len(g.pendingDecisions) == 0

	response := &Response{
		GameState:  gameState,
		Balance:    participant.balance,
		Hand:       participant.Hand(),
		CanDecide:  g.phase == PhaseDeclaration && g.pendingDecisions[playerID],
		HasDecided: g.phase == PhaseDeclaration && !g.pendingDecisions[playerID] && g.idToParticipant[playerID] != nil,
	}

	// Only show the player their decision after all have decided
	if allDecided {
		if decision, ok := g.decisions[playerID]; ok {
			response.MyDecision = &decision
		}
	}

	return &playable.Response{
		Key:   "game",
		Value: "guts",
		Data:  response,
	}, nil
}
