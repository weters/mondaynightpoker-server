package littlel

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
)

// Participant represents an individual participant in little L
type Participant struct {
	PlayerID   int64
	tableStake int

	didFold bool
	balance int
	hand    deck.Hand
	traded  int

	// currentBet is how much the player has bet in the current round
	currentBet int

	bestHand    *BestHand
	bestHandKey string
}

func newParticipant(id int64, tableStake int) *Participant {
	return &Participant{
		PlayerID:   id,
		tableStake: tableStake,
		didFold:    false,
		balance:    0,
		hand:       make(deck.Hand, 0),
	}
}

// reset is called after the round is reset
func (p *Participant) reset() {
	p.currentBet = 0
}

// BestHand is the best hand between the possible combinations
type BestHand struct {
	hand     deck.Hand
	analyzer *handanalyzer.HandAnalyzer
}

// GetBestHand returns the best hand the player can make with the exposed community cards
func (p *Participant) GetBestHand(community []*deck.Card) *BestHand {
	key := deck.CardsToString(p.hand) + deck.CardsToString(community)
	if p.bestHandKey == key {
		return p.bestHand
	}

	p.bestHand = p.getBestHand(community)
	p.bestHandKey = key
	return p.bestHand
}

func (p *Participant) getBestHand(community []*deck.Card) *BestHand {
	if len(community) != 3 {
		panic("invalid community")
	}

	hand1 := make([]*deck.Card, len(p.hand))
	copy(hand1, p.hand)

	hand2 := make([]*deck.Card, len(p.hand))
	copy(hand2, p.hand)

	if community[0] != nil {
		hand1 = append(hand1, community[0])

		if community[1] != nil {
			hand2 = append(hand2, community[1])

			if community[2] != nil {
				hand1 = append(hand1, community[2])
				hand2 = append(hand2, community[2])
			}
		}
	}

	ha1 := handanalyzer.New(3, hand1)
	if community[1] == nil {
		return &BestHand{
			hand:     hand1,
			analyzer: ha1,
		}
	}

	ha2 := handanalyzer.New(3, hand2)
	if ha1.GetStrength() > ha2.GetStrength() {
		return &BestHand{
			hand:     hand1,
			analyzer: ha1,
		}
	}

	return &BestHand{
		hand:     hand2,
		analyzer: ha2,
	}
}

// potmanager.Participant implementation

// ID returns the ID
func (p *Participant) ID() int64 {
	return p.PlayerID
}

// Balance returns the current balance
func (p *Participant) Balance() int {
	return p.balance + p.tableStake
}

// AdjustBalance will adjust the participant's balance
func (p *Participant) AdjustBalance(amount int) {
	p.balance += amount
}

// SetAmountInPlay sets the amount currently in play
func (p *Participant) SetAmountInPlay(amount int) {
	p.currentBet = amount
}
