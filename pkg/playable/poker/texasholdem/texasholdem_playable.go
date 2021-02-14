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

	var foundAction Action
	for _, a := range actions {
		if a.Name == action.Name {
			foundAction = a
			break
		}
	}

	p := g.participants[playerID]

	if foundAction.IsZero() {
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
		// this method handles nextDecision()
		g.participantFolded(p)
	}

	g.lastAction = &lastAction{
		Action:   action,
		PlayerID: p.PlayerID,
	}

	g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} %s", foundAction.LogString())
	return playable.OK(), true, nil
}

func (g *Game) participantFolded(p *Participant) {
	p.folded = true

	stillLive := 0
	for _, par := range g.participants {
		if !par.folded {
			stillLive++
		}

		if stillLive >= 2 {
			break
		}
	}

	// game is still going on
	if stillLive >= 2 {
		g.nextDecision()
		return
	}

	// not enough players left. end the game early
	g.setPendingDealerState(DealerStateRevealWinner, time.Second*2)
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
		balanceAdjustments[id] = player.balance
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: balanceAdjustments,
		Log:                g.gameLog(),
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
	for pos, winner := range winners {
		winner.won(g.getShareOfWinnings(n, pos))
	}

	logs := make([]*playable.LogMessage, 0, len(g.participantOrder))
	for _, pid := range g.participantOrder {
		p := g.participants[pid]

		hand := p.getHandAnalyzer(g.community).GetHand().String()
		if p.result == resultWon {
			logs = append(logs, playable.SimpleLogMessage(pid, "{} won ${%d} (${%d}) with a %s", p.winnings, p.balance, hand))
		} else if p.folded {
			logs = append(logs, playable.SimpleLogMessage(pid, "{} folded and lost ${%d}", -1*p.balance))
		} else {
			logs = append(logs, playable.SimpleLogMessage(pid, "{} lost ${%d} with a %s", -1*p.balance, hand))
		}
	}

	g.logChan <- logs
	g.setPendingDealerState(DealerStateEnd, time.Second*5)
	return nil
}

func (g *Game) getShareOfWinnings(winners, position int) int {
	if position >= winners {
		panic("position is out of range")
	}

	roundedWinnings := (g.pot / 25 / winners) * 25
	if position < (g.pot/25)%winners {
		roundedWinnings += 25
	}

	return roundedWinnings
}
