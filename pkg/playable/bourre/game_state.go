package bourre

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
)

// GameState is the overall game state
// This is safe for all players to see
type GameState struct {
	Seed          int64                `json:"seed"`
	Players       []*GameStatePlayer   `json:"players"`
	FoldedPlayers []*GameStatePlayer   `json:"foldedPlayers"`
	TrumpCard     *deck.Card           `json:"trumpCard"`
	PlayedCards   map[int64]*deck.Card `json:"playedCards"`
	CardsInDeck   int                  `json:"cardsInDeck"`
	Ante          int                  `json:"ante"`
	Pot           int                  `json:"pot"`
	Round         int                  `json:"round"`
	IsRoundOver   bool                 `json:"isRoundOver"`
	IsGameOver    bool                 `json:"isGameOver"`
	CurrentTurn   int64                `json:"currentTurn"`
}

// GameStatePlayer is the state of an individual player
// This is safe for all players to see
type GameStatePlayer struct {
	Balance        int   `json:"balance"`
	PlayerID       int64 `json:"playerId"`
	CardsInHand    int   `json:"cardsInHand"`
	CardsDiscarded int   `json:"cardsDiscarded"`
	// Decided determines if the player chose in/out
	Decided   bool `json:"decided"`
	Folded    bool `json:"folded"`
	TricksWon int  `json:"tricksWon"`
}

// Response is the response format for this game
type Response struct {
	GameState *GameState `json:"gameState"`
	// Data below is player specific, and must only be shown to the intended player
	Balance  int          `json:"balance"`
	Hand     []*deck.Card `json:"hand"`
	Discards []*deck.Card `json:"discards"`
	MaxDraw  int          `json:"maxDraw"`
	Folded   bool         `json:"folded"`
}

func (g *Game) getGameState() *GameState {
	players := make([]*GameStatePlayer, len(g.playerOrder))
	for player, i := range g.playerOrder {
		cards := g.playerDiscards[player]
		players[i] = &GameStatePlayer{
			Balance:        player.balance,
			PlayerID:       player.PlayerID,
			CardsInHand:    len(player.hand),
			CardsDiscarded: len(cards),
			Decided:        cards != nil || player.folded,
			Folded:         player.folded,
			TricksWon:      player.winCount,
		}
	}

	foldedPlayers := make([]*GameStatePlayer, 0)
	for player := range g.foldedPlayers {
		foldedPlayers = append(foldedPlayers, &GameStatePlayer{
			Balance:  player.balance,
			PlayerID: player.PlayerID,
			Decided:  true,
			Folded:   true,
		})
	}

	var currentTurn int64
	// if we have results, the game is over, thus no current turn
	if g.result == nil {
		if player := g.getCurrentTurn(); player != nil {
			currentTurn = player.PlayerID
		}
	}

	playedCards := make(map[int64]*deck.Card)
	for _, pc := range g.cardsPlayed {
		playedCards[pc.player.PlayerID] = pc.card
	}

	return &GameState{
		Seed:          g.deck.Seed(),
		Players:       players,
		FoldedPlayers: foldedPlayers,
		TrumpCard:     g.trumpCard,
		PlayedCards:   playedCards,
		CardsInDeck:   g.deck.CardsLeft(),
		Ante:          g.ante,
		Pot:           g.pot,
		Round:         g.roundNo,
		IsRoundOver:   g.isRoundOver(),
		IsGameOver:    g.result != nil,
		CurrentTurn:   currentTurn,
	}
}

// GetPlayerState returns the state for the given player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	player, ok := g.idToPlayer[playerID]
	if !ok {
		player = &Player{
			PlayerID: playerID,
			balance:  0,
			hand:     nil,
			folded:   true,
			winCount: 0,
		}
	}

	gameState := g.getGameState()
	maxDraw := 0
	if gameState.Round == 0 {
		maxDraw = g.maxDraw(player)
	}

	return &playable.Response{
		Key:   "game",
		Value: "bourre",
		Data: &Response{
			GameState: gameState,
			Balance:   player.balance,
			Hand:      player.hand,
			Discards:  g.playerDiscards[player],
			MaxDraw:   maxDraw,
			Folded:    player.folded,
		},
	}, nil
}
