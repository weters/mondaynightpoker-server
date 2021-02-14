package texasholdem

import (
	"fmt"
	"mondaynightpoker-server/pkg/playable"
	"time"
)

// Action performs a player action
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	actions := g.ActionsForParticipant(playerID)
	action, err := newAction(message.Action, 0)
	if err != nil {
		return nil, false, err
	}

	validAction := false
	for _, a := range actions {
		if a.Name == action.Name {
			validAction = true
			break
		}
	}

	p := g.participants[playerID]

	if !validAction {
		return nil, false, fmt.Errorf("you cannot perform %s", message.Action)
	}

	switch action.Name {
	case checkKey:
		g.nextDecision()
	case callKey:
		g.pot += p.Bet(g.currentBet)
		g.nextDecision()
	case betKey:
		amt, err := g.GetBetAmount()
		if err != nil {
			return nil, false, err
		}

		g.currentBet = amt
		g.pot += p.Bet(g.currentBet)
		g.setDecisionStartToCurrentTurn()
		g.nextDecision()
	case raiseKey:
		amt, err := g.GetBetAmount()
		if err != nil {
			return nil, false, err
		}

		g.currentBet += amt
		g.pot += p.Bet(g.currentBet)
		g.setDecisionStartToCurrentTurn()
		g.nextDecision()
	case foldKey:
		p.folded = true
		g.nextDecision()
	}

	return playable.OK(), true, nil
}

// GetPlayerState returns the current state for the player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	ps := g.getParticipantStateByPlayerID(playerID)
	return &playable.Response{
		Key:   "game",
		Value: g.Key(),
		Data:  ps,
	}, nil
}

// GetEndOfGameDetails returns details after the game finishes
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	if !g.finished {
		return nil, false
	}

	balanceAdjustments := make(map[int64]int)
	for id, player := range g.participants {
		balanceAdjustments[id] = player.Balance
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: balanceAdjustments,
		Log:                g,
	}, true
}

// Name returns the name
func (g *Game) Name() string {
	return NameFromOptions(g.options)
}

// NameFromOptions returns the name from the provided options
func NameFromOptions(opts Options) string {
	if err := validateOptions(opts); err != nil {
		return ""
	}

	return fmt.Sprintf("Limit Texas Hold'em (${%d}/${%d})", opts.LowerLimit, opts.LowerLimit*2)
}

// LogChan returns a channel log messages must be sent on
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}

// Key returns the key
func (g *Game) Key() string {
	return "texas-hold-em"
}

func (g *Game) setDecisionStartToCurrentTurn() {
	g.decisionStart = (g.decisionStart + g.decisionIndex) % len(g.participants)
	g.decisionIndex = 0
}

func (g *Game) endGame() error {
	if g.dealerState != DealerStateRevealWinner {
		return fmt.Errorf("cannot endGame from state %d", g.dealerState)
	}

	var bestHand int
	var winners []*Participant

	for _, p := range g.participants {
		if p.folded {
			p.result = resultFolded
			continue
		}

		p.result = resultLost
		p.reveal = true
		if s := p.getHandAnalyzer(g.community).GetStrength(); s > bestHand {
			bestHand = s
			winners = []*Participant{p}
		} else if s == bestHand {
			winners = append(winners, p)
		}
	}

	n := len(winners)
	for i, winner := range winners {
		winner.result = resultWon

		roundedWinnings := (g.pot / 25 / n) * 25
		if i < (g.pot/25)%n {
			roundedWinnings += 25
		}

		winner.Balance += roundedWinnings
	}

	g.setPendingDealerState(DealerStateEnd, time.Second*5)
	return nil
}
