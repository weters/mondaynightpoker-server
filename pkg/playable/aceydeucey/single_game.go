package aceydeucey

import (
	"github.com/google/uuid"
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
	UUID       string           `json:"uuid"`
	FirstCard  *deck.Card       `json:"firstCard"`
	MiddleCard *deck.Card       `json:"middleCard"`
	LastCard   *deck.Card       `json:"lastCard"`
	Bet        Bet              `json:"bet"`
	Adjustment int              `json:"adjustment"`
	Result     SingleGameResult `json:"result"`

	// isGameOver allows you to short-circuit the game over (i.e., free game)
	gameOver bool
}

// SingleGameResult is the result of a single game
type SingleGameResult string

// SingleGameResult constants
const (
	SingleGameResultFreeGame SingleGameResult = "free-game"
	SingleGameResultLost     SingleGameResult = "lost"
	SingleGameResultPost     SingleGameResult = "post"
	SingleGameResultWon      SingleGameResult = "won"
)

func newSingleGame() *SingleGame {
	return &SingleGame{
		UUID:       uuid.New().String(),
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
