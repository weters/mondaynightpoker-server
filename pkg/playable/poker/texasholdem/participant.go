package texasholdem

import (
	"mondaynightpoker-server/pkg/deck"
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

	balance int
	cards   deck.Hand

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
	Bet      int       `json:"bet"`
	Hand     string    `json:"hand"`
	Result   result    `json:"result"`
	Winnings int       `json:"winnings"`
}

func newParticipant(id int64) *Participant {
	return &Participant{
		PlayerID: id,
		balance:  0,
		cards:    make(deck.Hand, 0),
		result:   resultPending,
	}
}

// SubtractBalance subtracts from the player's balance
func (p *Participant) SubtractBalance(amount int) {
	p.balance -= amount
}

// FutureActionsForParticipant are actions the participant can perform when they are on the clock
func (g *Game) FutureActionsForParticipant(id int64) []Action {
	if !g.isParticipantPendingTurn(id) {
		return nil
	}

	p := g.participants[id]

	bet, err := g.GetBetAmount()
	if err != nil {
		panic(err)
	}

	futureActions := make([]Action, 0)
	if g.currentBet == 0 {
		return append(futureActions, actionCheck, mustAction(newAction(betKey, bet)), actionFold)
	}

	if g.currentBet > p.bet {
		// must call
		futureActions = append(futureActions, mustAction(newAction(callKey, g.currentBet-p.bet)))
	} else {
		// this should _only_ happen in opening round
		futureActions = append(futureActions, actionCheck)
	}

	if g.CanBet() {
		futureActions = append(futureActions, mustAction(newAction(raiseKey, g.currentBet+bet)))
	}

	futureActions = append(futureActions, actionFold)

	return futureActions
}

// isParticipantPendingTurn returns true if they still have to make a decision this turn
func (g *Game) isParticipantPendingTurn(id int64) bool {
	if !g.InBettingRound() {
		return false
	}

	var index = -1
	for i, pid := range g.participantOrder {
		if pid == id {
			index = i
			break
		}
	}

	if index < 0 {
		panic("could not find participant")
	}

	index -= g.decisionStart
	if index < 0 {
		index += len(g.participantOrder)
	}

	return index > g.decisionIndex
}

// ActionsForParticipant return the actions the current participant can take
func (g *Game) ActionsForParticipant(id int64) []Action {
	if !g.InBettingRound() {
		return nil
	}

	turn, err := g.GetCurrentTurn()
	if err != nil {
		panic(err)
	}

	if turn.PlayerID != id {
		return nil
	}

	actions := make([]Action, 0)
	if g.currentBet == turn.bet {
		actions = append(actions, actionCheck)
	} else if turn.bet < g.currentBet {
		call, _ := newAction(callKey, g.currentBet-turn.bet)
		actions = append(actions, call)
	}

	if g.CanBet() {
		amt, err := g.GetBetAmount()
		if err != nil {
			panic("could not get the bet amount")
		}

		if g.currentBet == 0 {
			bet, _ := newAction(betKey, amt)
			actions = append(actions, bet)
		} else {
			raise, _ := newAction(raiseKey, g.currentBet+amt)
			actions = append(actions, raise)
		}
	}

	return append(actions, actionFold)
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

func (p *Participant) won(amount int) {
	p.result = resultWon
	p.balance += amount
	p.winnings = amount
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
		Balance:  p.balance,
		Cards:    cards,
		Folded:   p.folded,
		Bet:      p.bet,
		Hand:     hand,
		Result:   p.result,
		Winnings: p.winnings,
	}
}
