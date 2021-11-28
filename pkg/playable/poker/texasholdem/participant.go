package texasholdem

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
)

type result string

const (
	resultPending result = ""
	resultFolded  result = "folded"
	resultLost    result = "lost"
	resultWon     result = "won"
)

// Participant represents an individual player in Texas Hold'em
type Participant struct {
	PlayerID int64

	balance    int
	tableStake int
	cards      deck.Hand

	folded bool
	reveal bool
	bet    int

	result   result
	winnings int

	handAnalyzer         *handanalyzer.HandAnalyzer
	handAnalyzerCacheKey string
}

type participantJSON struct {
	PlayerID int64     `json:"playerId"`
	Balance  int       `json:"balance"`
	Cards    deck.Hand `json:"cards"`
	Folded   bool      `json:"folded"`
	Bet      int       `json:"currentBet"`
	MinBet   int       `json:"minBet"`
	MaxBet   int       `json:"maxBet"`
	HandRank string    `json:"handRank"`
	Result   result    `json:"result"`
	Winnings int       `json:"winnings"`
}

func newParticipant(id int64, tableStake int) *Participant {
	return &Participant{
		PlayerID:   id,
		balance:    0,
		tableStake: tableStake,
		cards:      make(deck.Hand, 0),
		result:     resultPending,
	}
}

// FutureActionsForParticipant are actions the participant can perform when they are on the clock
func (g *Game) FutureActionsForParticipant(id int64) []action.Action {
	p := g.participants[id]
	if !g.potManager.IsParticipantYetToAct(p) {
		return nil
	}

	if g.dealerState >= DealerStateRevealWinner {
		return nil
	}

	if g.dealerState == DealerStateDiscardRound {
		return []action.Action{action.Discard}
	}

	currentBet := g.potManager.GetBet()

	if currentBet == p.bet {
		return []action.Action{action.Check, action.Fold}
	}

	return []action.Action{action.Call, action.Fold}
}

// ActionsForParticipant return the actions the current participant can take
func (g *Game) ActionsForParticipant(id int64) []action.Action {
	turn, err := g.GetCurrentTurn()
	if err != nil {
		return nil
	}

	if turn.PlayerID != id {
		return nil
	}

	if g.dealerState == DealerStateDiscardRound {
		return []action.Action{action.Discard}
	}

	currentBet := g.potManager.GetBet()

	actions := make([]action.Action, 0)
	if currentBet == turn.bet {
		actions = append(actions, action.Check)
	} else if turn.bet < currentBet {
		actions = append(actions, action.Call)
	}

	if currentBet == 0 {
		actions = append(actions, action.Bet)
	} else if g.potManager.GetParticipantAllInAmount(turn) > currentBet {
		actions = append(actions, action.Raise)
	}

	return append(actions, action.Fold)
}

// NewRound will reset the participant for a new round
func (p *Participant) NewRound() {
	p.bet = 0
}

func (p *Participant) getHandAnalyzer(community []*deck.Card) *handanalyzer.HandAnalyzer {
	if len(p.cards) == 0 {
		return nil
	}

	hand := append(p.cards, community...)
	key := hand.String()
	if p.handAnalyzerCacheKey != key {
		p.handAnalyzer = handanalyzer.New(5, hand)
		p.handAnalyzerCacheKey = key
	}

	return p.handAnalyzer
}

func (p *Participant) participantJSON(game *Game, forceReveal bool) *participantJSON {
	var cards deck.Hand
	var handRank string
	if forceReveal || (p.reveal && !p.folded) {
		cards = p.cards

		if ha := p.getHandAnalyzer(game.community); ha != nil {
			handRank = ha.GetHand().String()
		}
	} else {
		// make a null hand
		cards = make(deck.Hand, len(p.cards))
	}

	return &participantJSON{
		PlayerID: p.PlayerID,
		Balance:  p.Balance(),
		Cards:    cards,
		Folded:   p.folded,
		Bet:      p.bet,
		HandRank: handRank,
		Result:   p.result,
		Winnings: p.winnings,
	}
}

// potmanager.Participant interface

// ID returns the ID
func (p *Participant) ID() int64 {
	return p.PlayerID
}

// Balance returns the balance
func (p *Participant) Balance() int {
	return p.balance + p.tableStake
}

// AdjustBalance adjusts the balance
func (p *Participant) AdjustBalance(amount int) {
	p.balance += amount
}

// SetAmountInPlay sets the amount in play
func (p *Participant) SetAmountInPlay(amount int) {
	p.bet = amount
}
