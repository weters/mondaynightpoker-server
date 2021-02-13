package texasholdem

import (
	"encoding/json"
	"mondaynightpoker-server/pkg/deck"
)

// Participant represents an individual player in Texas Hold'em
type Participant struct {
	PlayerID int64
	Balance  int
	cards    deck.Hand

	folded bool
	reveal bool
	bet    int
}

// MarshalJSON handles custom marshalling of the JSON
func (p *Participant) MarshalJSON() ([]byte, error) {
	var cards deck.Hand
	if p.reveal && !p.folded {
		cards = p.cards
	}

	return json.Marshal(participantJSON{
		PlayerID: p.PlayerID,
		Balance:  p.Balance,
		Cards:    cards,
		Folded:   p.folded,
		Bet:      p.bet,
	})
}

type participantJSON struct {
	PlayerID int64     `json:"playerId"`
	Balance  int       `json:"balance"`
	Cards    deck.Hand `json:"cards"`
	Folded   bool      `json:"folded"`
	Bet      int       `json:"bet"`
}

func newParticipant(id int64) *Participant {
	return &Participant{
		PlayerID: id,
		Balance:  0,
		cards:    make(deck.Hand, 0),
	}
}

// SubtractBalance subtracts from the player's balance
func (p *Participant) SubtractBalance(amount int) {
	p.Balance -= amount
}

// ActionsForParticipant return the actions the current participant can take
func (g *Game) ActionsForParticipant(id int64) []Action {
	if !g.InBettingRound() {
		return nil
	}

	turn, err := g.GetCurrentTurn()
	if err != nil {
		panic("expected to have a participant")
	}

	if turn.PlayerID != id {
		return nil
	}

	actions := make([]Action, 0)
	if g.currentBet == turn.bet {
		actions = append(actions, ActionCheck)
	} else if turn.bet < g.currentBet {
		actions = append(actions, ActionCall)
	}

	if g.CanBet() {
		if g.currentBet == 0 {
			actions = append(actions, ActionBet)
		} else {
			actions = append(actions, ActionRaise)
		}
	}

	return append(actions, ActionFold)
}

// NewRound will reset the participant for a new round
func (g *Game) NewRound() {
	g.currentBet = 0
}
