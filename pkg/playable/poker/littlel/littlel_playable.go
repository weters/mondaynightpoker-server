package littlel

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
)

// --- Playable Interface ---

// Action performs a game action on behalf of the player
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	p, ok := g.idToParticipant[playerID]
	if !ok {
		return nil, false, errors.New("participant is not in the game")
	}

	switch message.Action {
	case "trade":
		if err := g.tradeCardsForParticipant(p, message.Cards); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	}

	return nil, false, fmt.Errorf("unknown action: %s", message.Action)
}

// GetPlayerState returns the state of the player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	p := g.idToParticipant[playerID]

	var action int64 = 0
	if currentTurn := g.GetCurrentTurn(); currentTurn != nil {
		action = currentTurn.PlayerID
	}

	s := State{
		Participant: &participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.balance,
			CurrentBet: p.currentBet,
			Hand:       p.hand,
			HandRank:   handanalyzer.New(3, p.hand).GetHand().String(), // nolint: FIXME
		},
		GameState: &GameState{
			Participants: make([]*participantJSON, 0),
			Stage:        g.stage,
			Action:       action,
			Pot:          g.pot,
			CurrentBet:   g.currentBet,
			TradeIns:     g.GetAllowedTradeIns(),
			InitialDeal:  g.options.InitialDeal,
			Community:    g.GetCommunityCards(),
		},
		Actions: g.getActionsForPlayer(playerID),
	}

	for _, id := range g.playerIDs {
		p := g.idToParticipant[id]
		pJSON := participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.balance,
			CurrentBet: p.currentBet,
		}

		if g.CanRevealCards() {
			pJSON.Hand = p.hand
		}

		s.GameState.Participants = append(s.GameState.Participants, &pJSON)
	}

	return &playable.Response{
		Key:   "game",
		Value: "little-l",
		Data:  &s,
	}, nil
}

// GetEndOfGameDetails returns the details at the end of a game
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	return nil, false
}

// Name returns the name of the game
func (g *Game) Name() string {
	return "Little L"
}

// LogChan returns a channel that can receive log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}
