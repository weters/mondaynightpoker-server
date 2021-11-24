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

	switch Action(message.Action) {
	case ActionTrade:
		if err := g.tradeCardsForParticipant(p, message.Cards); err != nil {
			return nil, false, err
		}

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} traded %d", len(message.Cards))

		return playable.OK(), true, nil
	case ActionCheck:
		if err := g.ParticipantChecks(p); err != nil {
			return nil, false, err
		}

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} checks")

		return playable.OK(), true, nil
	case ActionFold:
		if err := g.ParticipantFolds(p); err != nil {
			return nil, false, err
		}

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} folds")

		return playable.OK(), true, nil
	case ActionCall:
		if err := g.ParticipantCalls(p); err != nil {
			return nil, false, err
		}

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} calls")

		return playable.OK(), true, nil
	case ActionRaise:
		fallthrough
	case ActionBet:
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
		minBet := g.getMinBet()

		maxBet := g.potManager.GetPotLimitMaxBet()
		if allInAmount := g.potManager.GetParticipantAllInAmount(p); allInAmount < maxBet {
			maxBet = allInAmount
		}

		pJSON = &participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.Balance(),
			CurrentBet: p.currentBet,
			MinBet:     minBet,
			MaxBet:     maxBet,
			Hand:       p.hand,
			HandRank:   p.GetBestHand(g.GetCommunityCards()).analyzer.GetHand().String(),
		}
	}

	var winners map[int64]int
	if len(g.winners) > 0 {
		winners = make(map[int64]int)
		for pt, amt := range g.winners {
			winners[pt.PlayerID] = amt
		}
	}

	s := State{
		Participant: pJSON,
		GameState: &GameState{
			Name:         g.Name(),
			Participants: make([]*participantJSON, 0),
			DealerID:     g.idToParticipant[g.playerIDs[0]].PlayerID,
			Round:        g.round,
			Action:       action,
			Pot:          g.potManager.Pots().Total(),
			Pots:         g.potManager.Pots(),
			Ante:         g.options.Ante,
			CurrentBet:   g.potManager.GetBet(),
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
			Balance:    p.Balance(),
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

func (g *Game) getMinBet() int {
	minBet := g.options.Ante
	if currentBet := g.potManager.GetBet(); currentBet > 0 {
		minBet = currentBet + g.potManager.GetRaise()
	}
	return minBet
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
	name, _ := NameFromOptions(g.options)
	return name
}

// LogChan returns a channel that can receive log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}
