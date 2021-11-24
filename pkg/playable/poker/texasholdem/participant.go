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
	Hand     string    `json:"hand"`
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

// SubtractBalance subtracts from the player's balance
func (p *Participant) SubtractBalance(amount int) {
	p.balance -= amount
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

// Bet ensures the participant throws to the specified amount
// The value return is the amount added to the pot. For example, if a player already bet 100, and then calls
// for another 50, this method will return 50
func (p *Participant) Bet(amount int) int {
	diff := amount - p.bet
	p.bet = amount
	p.balance -= diff

	return diff
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
	var hand string
	if forceReveal || (p.reveal && !p.folded) {
		cards = p.cards

		if ha := p.getHandAnalyzer(game.community); ha != nil {
			hand = ha.GetHand().String()
		}
	}

	return &participantJSON{
		PlayerID: p.PlayerID,
		Balance:  p.Balance(),
		Cards:    cards,
		Folded:   p.folded,
		Bet:      p.bet,
		Hand:     hand,
		Result:   p.result,
		Winnings: p.winnings,
	}
}

// potmanager.Participant interface

func (p *Participant) ID() int64 {
	return p.PlayerID
}

func (p *Participant) Balance() int {
	return p.balance + p.tableStake
}

func (p *Participant) AdjustBalance(amount int) {
	p.balance += amount
}

func (p *Participant) SetAmountInPlay(amount int) {
	p.bet = amount
}
