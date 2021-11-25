package texasholdem

import (
	"fmt"
	"github.com/google/uuid"
	"math"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"mondaynightpoker-server/pkg/playable/poker/potmanager"
	"time"
)

// Action performs a player action
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	actions := g.ActionsForParticipant(playerID)

	anAction, err := action.FromString(message.Action)
	if err != nil {
		return nil, false, err
	}

	var foundAction action.Action
	for _, a := range actions {
		if a == anAction {
			foundAction = a
			break
		}
	}

	p := g.participants[playerID]

	if !foundAction.IsValid() {
		return nil, false, fmt.Errorf("you cannot perform %s", message.Action)
	}

	amount, _ := message.AdditionalData.GetInt("amount")

	switch foundAction {
	case action.Check:
		if err := g.potManager.ParticipantChecks(p); err != nil {
			return nil, false, err
		}
	case action.Call:
		if err := g.potManager.ParticipantCalls(p); err != nil {
			return nil, false, err
		}
	case action.Bet:
		fallthrough
	case action.Raise:
		if err := g.validateBetOrRaise(p, amount); err != nil {
			return nil, false, err
		}

		if err := g.potManager.ParticipantBetsOrRaises(p, amount); err != nil {
			return nil, false, err
		}

	case action.Fold:
		if err := g.potManager.ParticipantFolds(p); err != nil {
			return nil, false, err
		}

		p.folded = true
		if g.potManager.GetCanActParticipantCount() < 2 {
			// not enough players left. end the game early
			g.setPendingDealerState(DealerStateRevealWinner, time.Second*2)
		}
	}

	g.lastAction = &lastAction{
		Action:   foundAction,
		PlayerID: p.PlayerID,
		Amount:   amount,
	}

	if g.potManager.IsRoundOver() && g.pendingDealerState == nil {
		g.setPendingDealerState(DealerState(int(g.dealerState)+1), time.Second*1)
	}

	g.logChan <- playable.SimpleLogMessageSlice(p.PlayerID, "{} %s", foundAction.LogMessage(amount))
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

	var name string
	switch opts.Variant {
	case Standard:
		name = "Texas Hold'em"
	case Pineapple:
		name = "Pineapple"
	case LazyPineapple:
		name = "Lazy Pineapple"
	}

	return fmt.Sprintf("%s (${%d}/${%d})", name, opts.SmallBlind, opts.BigBlind)
}

// LogChan returns a channel log messages must be sent on
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}

// Key returns the key
func (g *Game) Key() string {
	return "texas-hold-em"
}

func (g *Game) endGame() error {
	if g.dealerState != DealerStateRevealWinner {
		return fmt.Errorf("cannot endGame from state %d", g.dealerState)
	}

	g.potManager.EndGame()

	wm := potmanager.NewWinManager()
	for _, p := range g.participantOrder {
		if p.folded {
			p.result = resultFolded
			continue
		}

		p.result = resultLost
		p.reveal = true

		strength := p.getHandAnalyzer(g.community).GetStrength()
		wm.AddParticipant(p, strength)
	}

	winners, err := g.potManager.PayWinners(wm.GetSortedTiers())
	if err != nil {
		return err
	}

	for p, amt := range winners {
		pt := g.participants[p.ID()]
		pt.result = resultWon
		pt.winnings = amt
	}

	logs := make([]*playable.LogMessage, 0, len(g.participantOrder))
	for _, p := range g.participantOrder {
		pid := p.ID()

		hand := p.getHandAnalyzer(g.community).GetHand().String()
		msg := playable.LogMessage{
			UUID:      uuid.New().String(),
			PlayerIDs: []int64{pid},
			Time:      time.Now(),
			Cards:     nil,
			Message:   "",
		}
		if p.result == resultWon {
			msg.Message = fmt.Sprintf("{} won ${%d} (${%d}) with a %s", p.winnings, p.balance, hand)
			msg.Cards = p.cards
		} else if p.folded {
			msg.Message = fmt.Sprintf("{} folded and lost ${%d}", -1*p.balance)
		} else {
			msg.Message = fmt.Sprintf("{} lost ${%d} with a %s", -1*p.balance, hand)
			msg.Cards = p.cards
		}

		logs = append(logs, &msg)
	}

	g.logChan <- logs
	g.setPendingDealerState(DealerStateEnd, time.Second*5)
	return nil
}

func (g *Game) validateBetOrRaise(p *Participant, amount int) error { // nolint:interfacer
	if amount%25 > 0 {
		return fmt.Errorf("bet must be in increments of ${25}")
	}

	potLimit := g.potManager.GetPotLimitMaxBet()
	allInAmont := g.potManager.GetParticipantAllInAmount(p)

	if currentBet := g.potManager.GetBet(); currentBet > 0 {
		if amount > potLimit {
			return fmt.Errorf("raise must not exceed total of ${%d}", potLimit)
		}

		if amount < currentBet {
			return fmt.Errorf("you cannot raise to an amount less than the current bet")
		}

		raise := g.potManager.GetRaise() + g.potManager.GetBet()
		if amount < allInAmont && amount < raise {
			return fmt.Errorf("raise must be to at least ${%d}", raise)
		}

		return nil
	}

	minBet := max(g.options.Ante, g.options.BigBlind, 25)
	if amount > potLimit {
		return fmt.Errorf("bet must be at most ${%d}", potLimit)
	} else if amount < allInAmont && amount < minBet {
		return fmt.Errorf("bet must be at least ${%d}", minBet)
	}

	return nil
}

func max(numbers ...int) int {
	max := math.MinInt
	for _, i := range numbers {
		if i > max {
			max = i
		}
	}

	return max
}
