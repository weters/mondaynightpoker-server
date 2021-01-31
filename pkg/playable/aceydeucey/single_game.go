package aceydeucey

import (
	"mondaynightpoker-server/pkg/deck"
)

// Bet represents a bet
type Bet struct {
	// Amount is the bet amount
	Amount int `json:"amount"`
	// HalfPot means that if they win, they win half the pot, not the amount
	HalfPot bool `json:"halfPot"`
}

// SingleGame is an individual game of Acey Deucey
type SingleGame struct {
	FirstCard  *deck.Card `json:"firstCard"`
	MiddleCard *deck.Card `json:"middleCard"`
	LastCard   *deck.Card `json:"lastCard"`
	Bet        Bet        `json:"bet"`
	Adjustment int        `json:"adjustment"`

	// isGameOver allows you to short-circuit the game over (i.e., free game)
	gameOver bool
}

func newSingleGame() *SingleGame {
	return &SingleGame{
		FirstCard:  nil,
		MiddleCard: nil,
		LastCard:   nil,
		Bet: Bet{
			Amount:  0,
			HalfPot: false,
		},
		gameOver: false,
	}
}

// firstCardRank will return the rank of the first card
// The first card may be a low-ace, so we'll check and handle that situation specifically.
func (g *SingleGame) firstCardRank() int {
	if g.FirstCard == nil {
		panic("FirstCard is not set")
	}

	if g.FirstCard.Rank == deck.Ace && g.FirstCard.IsBitSet(aceStateLow) {
		return deck.LowAce
	}

	return g.FirstCard.Rank
}

func (g *SingleGame) isGameOver() bool {
	return g.MiddleCard != nil || g.gameOver
}
