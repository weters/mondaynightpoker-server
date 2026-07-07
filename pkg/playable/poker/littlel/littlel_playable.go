package littlel

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker"
	"mondaynightpoker-server/pkg/playable/poker/action"
)

// --- Playable Interface ---

// Action performs a game action on behalf of the player
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	p, ok := g.idToParticipant[playerID]
	if !ok {
		return nil, false, errors.New("participant is not in the game")
	}

	logs := make([]*playable.LogMessage, 0, 2)
	canGoAllIn := false
	currentAction := action.Action(message.Action)
	amount := 0

	switch currentAction {
	case action.Trade:
		if err := g.tradeCardsForParticipant(p, message.Cards); err != nil {
			return nil, false, err
		}

		g.recordAction(action.Trade, p.PlayerID, 0, message.Cards, false)

		if len(message.Cards) == 0 {
			logs = append(logs, playable.SimpleLogMessage(p.PlayerID, "{} stands pat (no trades)"))
		} else {
			logs = append(logs, playable.SimpleLogMessage(p.PlayerID, "{} traded %d card(s)", len(message.Cards)))
		}

		g.SendLogMessages(logs)
		return playable.OK(), true, nil
	case action.Check:
		if err := g.ParticipantChecks(p); err != nil {
			return nil, false, err
		}

		logs = append(logs, playable.SimpleLogMessage(p.PlayerID, "{} checks"))
	case action.Fold:
		if err := g.ParticipantFolds(p); err != nil {
			return nil, false, err
		}

		logs = append(logs, playable.SimpleLogMessage(p.PlayerID, "{} folds"))
	case action.Call:
		if err := g.ParticipantCalls(p); err != nil {
			return nil, false, err
		}

		canGoAllIn = true
		logs = append(logs, playable.SimpleLogMessage(p.PlayerID, "{} calls"))
	case action.Raise:
		fallthrough
	case action.Bet:
		amount, _ = message.AdditionalData.GetInt("amount")
		if amount == 0 {
			return nil, false, errors.New("amount must be > 0")
		}

		if err := g.ParticipantBets(p, amount); err != nil {
			return nil, false, err
		}

		canGoAllIn = true
		if message.Action == "raise" {
			logs = append(logs, playable.SimpleLogMessage(p.PlayerID, "{} raises to ${%d}", amount))
		} else {
			logs = append(logs, playable.SimpleLogMessage(p.PlayerID, "{} bets ${%d}", amount))
		}
	default:
		return nil, false, fmt.Errorf("unknown action: %s", message.Action)
	}

	allIn := canGoAllIn && g.potManager.IsParticipantAllIn(p)
	if allIn {
		logs = append(logs, playable.SimpleLogMessage(p.PlayerID, "{} is all-in"))
	}

	g.recordAction(currentAction, p.PlayerID, amount, nil, allIn)

	g.SendLogMessages(logs)
	return playable.OK(), true, nil
}

// GetPlayerState returns the state of the player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	var action int64
	if currentTurn := g.GetCurrentTurn(); currentTurn != nil {
		action = currentTurn.PlayerID
	}

	p, ok := g.idToParticipant[playerID]
	var pJSON *participantJSON
	if ok {
		pJSON = &participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.Balance(),
			CurrentBet: p.currentBet,
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
			TradeIns:     g.GetAllowedTradeIns(),
			InitialDeal:  g.options.InitialDeal,
			Winners:      winners,
		},
		PokerState: &poker.State{
			Ante:       g.options.Ante,
			CurrentBet: g.potManager.GetBet(),
			MinBet:     g.getMinBet(),
			MaxBet:     g.potManager.GetPotLimitMaxBet(),
			Pots:       g.potManager.Pots(),
			Community:  g.GetCommunityCards(),
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

	balanceAdjustments := make(map[int64]int)
	for _, p := range g.idToParticipant {
		balanceAdjustments[p.PlayerID] = p.balance
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: balanceAdjustments,
		Log:                g.finalGameLog(),
	}, true
}

// Name returns the name of the game
func (g *Game) Name() string {
	name, _ := NameFromOptions(g.options)
	return name
}
