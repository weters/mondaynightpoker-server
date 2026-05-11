package sevencard

import (
	"errors"
	"mondaynightpoker-server/pkg/playable"
)

// Name returns the name of the game
func (g *Game) Name() string {
	return g.options.Variant.Name()
}

// Action performs a game action on behalf of the player
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	p, ok := g.idToParticipant[playerID]
	if !ok {
		return nil, false, errors.New("you are not in the game")
	}

	// Check for variant-specific actions first
	if iv, ok := g.options.Variant.(InteractiveVariant); ok {
		variantAction := Action(message.Action)
		handled, err := iv.HandleVariantAction(g, p, variantAction)
		if err != nil {
			return nil, false, err
		}
		if handled {
			g.recordAction(variantAction, p.PlayerID, 0, nil, false)
			if len(g.pendingLogs) > 0 {
				g.logChan <- g.pendingLogs
				g.pendingLogs = make([]*playable.LogMessage, 0)
			}
			return playable.OK(), true, nil
		}
	}

	action, err := ActionFromString(message.Action)
	if err != nil {
		return nil, false, err
	}

	canGoAllIn := false
	amount, _ := message.AdditionalData.GetInt("amount")
	switch action {
	case ActionCheck:
		if err := g.participantChecks(p); err != nil {
			return nil, false, err
		}

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} checks")
	case ActionBet:
		if amount <= 0 {
			return nil, false, errors.New("invalid amount")
		}

		if err := g.participantBets(p, amount); err != nil {
			return nil, false, err
		}

		canGoAllIn = true
		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} bets ${%d}", amount)
	case ActionRaise:
		if amount <= 0 {
			return nil, false, errors.New("invalid amount")
		}

		if err := g.participantRaises(p, amount); err != nil {
			return nil, false, err
		}

		canGoAllIn = true
		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} raises to ${%d}", amount)
	case ActionCall:
		if err := g.participantCalls(p); err != nil {
			return nil, false, err
		}

		canGoAllIn = true
		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} calls")
	case ActionFold:
		if err := g.participantFolds(p); err != nil {
			return nil, false, err
		}

		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} folds")
	}

	allIn := canGoAllIn && p.isAllIn()
	if allIn {
		g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} is all-in")
	}

	g.recordAction(action, p.PlayerID, amount, nil, allIn)

	// Track the last action for UI feedback
	g.lastAction = &lastAction{
		PlayerID: playerID,
		Action:   action,
	}

	if len(g.pendingLogs) > 0 {
		g.logChan <- g.pendingLogs
		g.pendingLogs = make([]*playable.LogMessage, 0)
	}

	return playable.OK(), true, nil
}

// GetPlayerState returns the player and game state for the specified player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	return &playable.Response{
		Key:   "game",
		Value: "seven-card",
		Data:  g.getPlayerStateByPlayerID(playerID),
	}, nil
}

// GetEndOfGameDetails returns details about the end of the game if the game is over
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

// LogChan returns a channel where another goroutine can listen for log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}
