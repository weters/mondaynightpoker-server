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
