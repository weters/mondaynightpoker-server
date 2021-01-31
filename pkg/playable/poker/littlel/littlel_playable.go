package littlel

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
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

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} traded %d", len(message.Cards))

		return playable.OK(), true, nil
	case "next-round":
		if err := g.NextRound(); err != nil {
			return nil, false, err
		}

		if !g.IsGameOver() {
			g.logChan <- playable.SimpleLogMessageSlice(0, "Next round started")
		}

		return playable.OK(), true, nil
	case "check":
		if err := g.ParticipantChecks(p); err != nil {
			return nil, false, err
		}

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} checks")

		return playable.OK(), true, nil
	case "fold":
		if err := g.ParticipantFolds(p); err != nil {
			return nil, false, err
		}

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} folds")

		return playable.OK(), true, nil
	case "call":
		if err := g.ParticipantCalls(p); err != nil {
			return nil, false, err
		}

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} calls")

		return playable.OK(), true, nil
	case "raise":
		fallthrough
	case "bet":
		amount, _ := message.AdditionalData.GetInt("amount")
		if amount == 0 {
			return nil, false, errors.New("amount must be > 0")
		}

		if err := g.ParticipantBets(p, amount); err != nil {
			return nil, false, err
		}

		if message.Action == "raise" {
			g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} raises to ${%d}", amount)
		} else {
			g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} bets ${%d}", amount)
		}

		return playable.OK(), true, nil
	case "end-game":
		if !g.IsGameOver() {
			return nil, false, errors.New("game is not over")
		}

		g.done = true
		return playable.OK(), true, nil
	}

	return nil, false, fmt.Errorf("unknown action: %s", message.Action)
}

// GetPlayerState returns the state of the player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	var action int64 = 0
	if currentTurn := g.GetCurrentTurn(); currentTurn != nil {
		action = currentTurn.PlayerID
	}

	p, ok := g.idToParticipant[playerID]
	var pJSON *participantJSON
	if ok {
		pJSON = &participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.balance,
			CurrentBet: p.currentBet,
			Hand:       p.hand,
			HandRank:   p.GetBestHand(g.GetCommunityCards()).analyzer.GetHand().String(),
		}
	}

	winners := make([]int64, len(g.winners))
	for i, p := range g.winners {
		winners[i] = p.PlayerID
	}

	s := State{
		Participant: pJSON,
		GameState: &GameState{
			Participants: make([]*participantJSON, 0),
			DealerID:     g.idToParticipant[g.playerIDs[0]].PlayerID,
			Round:        g.round,
			Action:       action,
			Pot:          g.pot,
			Ante:         g.options.Ante,
			CurrentBet:   g.currentBet,
			MaxBet:       g.getPotLimit(),
			TradeIns:     g.GetAllowedTradeIns(),
			InitialDeal:  g.options.InitialDeal,
			Community:    g.GetCommunityCards(),
			Winners:      winners,
		},
		Actions:       g.getActionsForPlayer(playerID),
		FutureActions: g.getFutureActionsForPlayer(playerID),
	}

	for _, id := range g.playerIDs {
		p := g.idToParticipant[id]
		pJSON := participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.balance,
			CurrentBet: p.currentBet,
			Traded:     p.traded,
		}

		if g.CanRevealCards() {
			if p.didFold {
				pJSON.HandRank = "Folded"
			} else {
				pJSON.Hand = p.hand
				pJSON.HandRank = p.GetBestHand(g.GetCommunityCards()).analyzer.GetHand().String()
			}
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
	if !g.done {
		return nil, false
	}

	c := g.GetCommunityCards()
	balanceAdjustments := make(map[int64]int)
	hands := make(map[int64]*participantJSON)
	for _, p := range g.idToParticipant {
		balanceAdjustments[p.PlayerID] = p.balance
		hands[p.PlayerID] = &participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.balance,
			CurrentBet: 0,
			Traded:     p.traded,
			Hand:       p.hand,
			HandRank:   p.GetBestHand(c).analyzer.GetHand().String(),
		}
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: balanceAdjustments,
		Log: struct {
			Hands     map[int64]*participantJSON
			Community deck.Hand
		}{
			Hands:     hands,
			Community: c,
		},
	}, true
}

// Name returns the name of the game
func (g *Game) Name() string {
	return "Little L"
}

// LogChan returns a channel that can receive log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}
