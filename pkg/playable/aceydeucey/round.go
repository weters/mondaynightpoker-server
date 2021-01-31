package aceydeucey

import (
	"errors"
	"math"
	"mondaynightpoker-server/pkg/deck"
)

var errorRoundIsOver = errors.New("round is over")

const (
	aceStateLow = 1 << iota
	aceStateHigh
)

// Round is a collection of one or more SingleGame
type Round struct {
	Games []*SingleGame `json:"games"`
	State RoundState    `json:"state"`
	Pot   int           `json:"pot"`

	activeGameIndex int
	deck            *deck.Deck
}

// RoundState is the state of the current round
type RoundState string

// RoundState constants
const (
	// roundStateStart is before any cards have been dealt
	RoundStateStart RoundState = "start"

	// roundStateFirstCard means only the first card has been dealt
	RoundStateFirstCardDealt RoundState = "first-card-dealt"

	// roundStatePendingAceDecision means the first card has been dealt, it's an ace, and the player needs to pick high/low
	RoundStatePendingAceDecision RoundState = "pending-ace-decision"

	// roundStatePendingBet means the last card has been dealt and we are waiting for participant to place bet
	RoundStatePendingBet RoundState = "pending-bet"

	// RoundStateBet means a bet has been successfully placed
	RoundStateBetPlaced RoundState = "bet-placed"

	// roundStateGameOver means the game ended and there's still at least one more game to be played
	RoundStateGameOver RoundState = "game-over"

	// roundStateRoundOver means all games have finished
	RoundStateRoundOver RoundState = "round-over"
)

// NewRound returns a new Round object
func NewRound(d *deck.Deck, startingPot int) *Round {
	return &Round{
		Games: []*SingleGame{newSingleGame()},
		State: RoundStateStart,
		Pot:   startingPot,

		activeGameIndex: 0,
		deck:            d,
	}
}

// DealCard deals a card in Acey Deucey
func (r *Round) DealCard() error {
	game, err := r.activeGame()
	if err != nil {
		return err
	}

	card, err := r.drawCard()
	if err != nil {
		return err
	}

	if game.FirstCard == nil {
		return r.dealFirstCard(card)
	}

	if game.LastCard == nil {
		return r.dealLastCard(card)
	}

	return r.dealMiddleCard(card)
}

func (r *Round) dealFirstCard(card *deck.Card) error {
	game := r.Games[r.activeGameIndex]
	game.FirstCard = card
	if card.Rank == deck.Ace {
		r.State = RoundStatePendingAceDecision
		return nil
	}

	r.State = RoundStateFirstCardDealt
	return nil
}

func (r *Round) dealLastCard(card *deck.Card) error {
	game, err := r.activeGame()
	if err != nil {
		return err
	}

	if card := game.FirstCard; card.Rank == deck.Ace {
		if !card.IsBitSet(aceStateLow) && !card.IsBitSet(aceStateHigh) {
			return errors.New("ace has not been decided")
		}
	}

	game.LastCard = card

	firstCardRank := game.firstCardRank()

	// same rank dealt, create another game
	if firstCardRank == card.Rank {
		game.LastCard = nil
		newGame := newSingleGame()
		newGame.FirstCard = card
		r.Games = append(r.Games, newGame)
		r.State = RoundStateFirstCardDealt
		return nil
	}

	if math.Abs(float64(card.Rank-firstCardRank)) == 1 {
		game.gameOver = true
		r.finalizeGame(game, 0)
		return nil
	}

	r.State = RoundStatePendingBet
	return nil
}

// finalizeGame updates balances and sets the state
func (r *Round) finalizeGame(g *SingleGame, adjustment int) {
	g.Adjustment = adjustment
	r.Pot -= adjustment

	// if this is the last game or the Pot is empty, end it
	if r.activeGameIndex+1 == len(r.Games) || r.Pot == 0 {
		r.State = RoundStateRoundOver
	} else {
		r.State = RoundStateGameOver
	}
}

func (r *Round) dealMiddleCard(card *deck.Card) error {
	game := r.Games[r.activeGameIndex]
	game.MiddleCard = card

	firstCardRank := game.firstCardRank()

	if card.Rank == firstCardRank || card.Rank == game.LastCard.Rank {
		r.finalizeGame(game, -2*game.Bet.Amount)
		return nil
	}

	lowCard, highCard := firstCardRank, game.LastCard.Rank
	if firstCardRank > game.LastCard.Rank {
		// swap them around
		lowCard, highCard = highCard, lowCard
	}

	if card.Rank > lowCard && card.Rank < highCard {
		if game.Bet.HalfPot {
			halfPot := r.Pot / 2
			halfPot -= halfPot % 25

			r.finalizeGame(game, halfPot)
		} else {
			r.finalizeGame(game, game.Bet.Amount)
		}

		return nil
	}

	r.finalizeGame(game, -1*game.Bet.Amount)
	return nil
}

// drawCard will draw a card and it should always succeed
func (r *Round) drawCard() (*deck.Card, error) {
	if !r.deck.CanDraw(1) {
		cards := r.getCardsInActiveGame()
		r.deck.Shuffle(seed)

		for _, card := range cards {
			r.deck.RemoveCard(card)
		}
	}

	return r.deck.Draw()
}

func (r *Round) isRoundOver() bool {
	return r.State == RoundStateRoundOver
}

func (r *Round) activeGame() (*SingleGame, error) {
	if r.isRoundOver() {
		return nil, errorRoundIsOver
	}

	activeGame := r.Games[r.activeGameIndex]
	if activeGame.isGameOver() {
		return nil, errors.New("game is over")
	}

	return activeGame, nil
}

func (r *Round) canBetTheGap() bool {
	game, err := r.activeGame()
	if err != nil {
		return false
	}

	if r.State != RoundStatePendingBet {
		return false
	}

	return math.Abs(float64(game.firstCardRank()-game.LastCard.Rank)) == 2
}

func (r *Round) setBet(bet int, isHalfPotBet bool) error {
	game, err := r.activeGame()
	if err != nil {
		return err
	}

	if game.LastCard == nil {
		return errors.New("cannot bet yet")
	}

	if game.Bet.Amount > 0 {
		return errors.New("bet has already been made")
	}

	if bet > r.Pot {
		return errors.New("bet must not exceed the Pot")
	}

	if isHalfPotBet {
		if !r.canBetTheGap() {
			return errors.New("bet the gap for half-pot requires a one-card gap")
		}
	}

	game.Bet = Bet{
		Amount:  bet,
		HalfPot: isHalfPotBet,
	}

	r.State = RoundStateBetPlaced
	return nil
}

func (r *Round) nextGame() error {
	if r.isRoundOver() {
		return errorRoundIsOver
	}

	game := r.Games[r.activeGameIndex]
	if !game.isGameOver() {
		return errors.New("game is not over")
	}

	r.activeGameIndex++
	if r.Games[r.activeGameIndex].FirstCard.Rank == deck.Ace {
		r.State = RoundStatePendingAceDecision
	} else {
		r.State = RoundStateFirstCardDealt
	}

	return nil
}

func (r *Round) setAce(highAce bool) error {
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

	r.State = RoundStateFirstCardDealt
	card.SetBit(bit)
	return nil
}

// getCardsInActiveGame will return cards that are still in an active game
// The intent for this method is to handle end-of-deck scenarios where some cards
// have been dealt already
func (r *Round) getCardsInActiveGame() []*deck.Card {
	if r.isRoundOver() {
		return nil
	}

	cards := make([]*deck.Card, 0)
	for i := r.activeGameIndex; i < len(r.Games); i++ {
		game := r.Games[i]
		if game.isGameOver() {
			continue
		}

		if game.FirstCard != nil {
			cards = append(cards, game.FirstCard)
		}

		if game.LastCard != nil {
			cards = append(cards, game.LastCard)
		}
	}

	if len(cards) == 0 {
		return nil
	}

	return cards
}

// ParticipantAdjustments returns the adjustments for the participant
func (r *Round) ParticipantAdjustments() int {
	adjustment := 0
	for _, game := range r.Games {
		adjustment += game.Adjustment
	}

	return adjustment
}
