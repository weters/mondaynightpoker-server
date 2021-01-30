package aceydeucey

import (
	"errors"
	"math"
	"mondaynightpoker-server/pkg/deck"
)

const (
	aceStateLow = 1 << iota
	aceStateHigh
)

// singleGame is an individual game of Acey Deucey
type singleGame struct {
	FirstCard, MiddleCard, LastCard *deck.Card
	Action                          Action
	Bet                             int

	// isGameOver allows you to short-circuit the game over (i.e., free game)
	isFreeGame bool
}

// round is a collection of one or more singleGame
type round struct {
	Games           []*singleGame
	ActiveGameIndex int
}

type roundState int

const (
	// roundStateStart is before any cards have been dealt
	roundStateStart roundState = iota

	// roundStateFirstCard means only the first card has been dealt
	roundStateFirstCard

	// roundStatePendingAceDecision means the first card has been dealt, it's an ace, and the player needs to pick high/low
	roundStatePendingAceDecision

	// roundStatePendingBet means the last card has been dealt and we are waiting for participant to place bet
	roundStatePendingBet

	// roundStateGameOver means the game ended and there's still at least one more game to be played
	roundStateGameOver

	// roundStateRoundOver means all games have finished
	roundStateRoundOver
)

// newRound returns a new round object
func newRound() *round {
	return &round{
		Games:           []*singleGame{newSingleGame()},
		ActiveGameIndex: 0,
	}
}

type addCardResponse int

const (
	addCardResponseFail addCardResponse = iota
	addCardResponseOK
	addCardResponseWaitingOnAce
	addCardResponseWaitingOnBet
	addCardResponseFreeGame
	addCardResponseDoubleGame
	addCardResponseWon
	addCardResponseLost
	addCardResponseLostPost
)

func (r *round) addFirstCard(card *deck.Card) (addCardResponse, error) {
	game := r.Games[r.ActiveGameIndex]
	game.FirstCard = card
	if card.Rank == deck.Ace {
		return addCardResponseWaitingOnAce, nil
	}

	return addCardResponseOK, nil
}

func (r *round) addLastCard(card *deck.Card) (addCardResponse, error) {
	game, err := r.activeGame()
	if err != nil {
		return 0, err
	}

	if card := game.FirstCard; card.Rank == deck.Ace {
		if !card.IsBitSet(aceStateLow) && !card.IsBitSet(aceStateHigh) {
			return addCardResponseFail, errors.New("ace has not been decided")
		}
	}

	game.LastCard = card

	firstCardRank := game.firstCardRank()
	if firstCardRank == card.Rank {
		game.LastCard = nil
		newGame := newSingleGame()
		newGame.FirstCard = card
		r.Games = append(r.Games, newGame)
		return addCardResponseDoubleGame, nil
	}

	if math.Abs(float64(card.Rank-firstCardRank)) == 1 {
		game.isFreeGame = true
		return addCardResponseFreeGame, nil
	}

	return addCardResponseWaitingOnBet, nil
}

func (r *round) addMiddleCard(card *deck.Card) (addCardResponse, error) {
	game := r.Games[r.ActiveGameIndex]
	game.MiddleCard = card

	firstCardRank := game.firstCardRank()

	if card.Rank == firstCardRank || card.Rank == game.LastCard.Rank {
		return addCardResponseLostPost, nil
	}

	lowCard, highCard := firstCardRank, game.LastCard.Rank
	if firstCardRank > game.LastCard.Rank {
		// swap them around
		lowCard, highCard = highCard, lowCard
	}

	if card.Rank > lowCard && card.Rank < highCard {
		return addCardResponseWon, nil
	}

	return addCardResponseLost, nil
}

func (r *round) addCard(card *deck.Card) (addCardResponse, error) {
	game, err := r.activeGame()
	if err != nil {
		return addCardResponseFail, err
	}

	if game.FirstCard == nil {
		return r.addFirstCard(card)
	}

	if game.LastCard == nil {
		return r.addLastCard(card)
	}

	return r.addMiddleCard(card)
}

func newSingleGame() *singleGame {
	return &singleGame{
		FirstCard:  nil,
		MiddleCard: nil,
		LastCard:   nil,
		Action:     ActionPending,
		Bet:        0,
	}
}

// firstCardRank will return the rank of the first card
// The first card may be a low-ace, so we'll check and handle that situation specifically.
func (g *singleGame) firstCardRank() int {
	if g.FirstCard == nil {
		panic("FirstCard is not set")
	}

	if g.FirstCard.Rank == deck.Ace && g.FirstCard.IsBitSet(aceStateLow) {
		return deck.LowAce
	}

	return g.FirstCard.Rank
}

func (g *singleGame) isGameOver() bool {
	return g.MiddleCard != nil || g.isFreeGame
}

func (r *round) isRoundOver() bool {
	for _, game := range r.Games {
		if !game.isGameOver() {
			return false
		}
	}

	return true
}

var errorRoundIsOver = errors.New("round is over")

func (r *round) activeGame() (*singleGame, error) {
	if r.isRoundOver() {
		return nil, errorRoundIsOver
	}

	activeGame := r.Games[r.ActiveGameIndex]
	if activeGame.isGameOver() {
		return nil, errors.New("game is over")
	}

	return activeGame, nil
}

func (r *round) nextGame() error {
	if r.isRoundOver() {
		return errorRoundIsOver
	}

	game := r.Games[r.ActiveGameIndex]
	if !game.isGameOver() {
		return errors.New("game is not over")
	}

	r.ActiveGameIndex++
	return nil
}

func (r *round) setAce(highAce bool) error {
	game, err := r.activeGame()
	if err != nil {
		return err
	}

	card := game.FirstCard
	if card.Rank != deck.Ace {
		return errors.New("first card is not an ace")
	}

	if card.IsBitSet(aceStateLow) || card.IsBitSet(aceStateHigh) {
		return errors.New("ace has already been decided")
	}

	bit := aceStateLow
	if highAce {
		bit = aceStateHigh
	}

	card.SetBit(bit)
	return nil
}

func (r *round) getState() roundState {
	if r.isRoundOver() {
		return roundStateRoundOver
	}

	activeGame := r.Games[r.ActiveGameIndex]
	if activeGame.isGameOver() {
		return roundStateGameOver
	}

	if activeGame.FirstCard == nil {
		return roundStateStart
	}

	if firstCard := activeGame.FirstCard; firstCard.Rank == deck.Ace && !firstCard.IsBitSet(aceStateLow) && !firstCard.IsBitSet(aceStateHigh) {
		return roundStatePendingAceDecision
	}

	if activeGame.LastCard == nil {
		return roundStateFirstCard
	}

	if activeGame.MiddleCard == nil {
		return roundStatePendingBet
	}

	panic("did not account for this state")
}
