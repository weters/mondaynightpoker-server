package aceydeucey

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"
)

var errorRoundIsOver = errors.New("round is over")

const (
	aceStateUndecided = 1 << iota
	aceStateLow
	aceStateHigh
)

// Round is a collection of one or more SingleGame
type Round struct {
	PlayerID int64
	Games    []*SingleGame
	State    RoundState
	Pot      int
	// HalfPotMax will limit the bet to half-pot if true
	HalfPotMax bool
	logChan    chan []*playable.LogMessage

	activeGameIndex int
	deck            *deck.Deck
	nextAction      *nextAction
}

// MarshalJSON provides custom JSON marshalling for round
func (r *Round) MarshalJSON() ([]byte, error) {
	return json.Marshal(roundJSON{
		Games:           r.Games,
		State:           r.State,
		Pot:             r.Pot,
		ActiveGameIndex: r.activeGameIndex,
		CardsRemaining:  r.deck.CardsLeft(),
	})
}

type roundJSON struct {
	Games           []*SingleGame `json:"games"`
	State           RoundState    `json:"state"`
	Pot             int           `json:"pot"`
	ActiveGameIndex int           `json:"activeGameIndex"`
	CardsRemaining  int           `json:"cardsRemaining"`
}

type nextAction struct {
	Time      time.Time
	NextState RoundState
}

// RoundState is the state of the current round
type RoundState string

// RoundState constants
const (
	// RoundStateStart is before any cards have been dealt
	RoundStateStart RoundState = "start"

	// RoundStateFirstCard means only the first card has been dealt
	RoundStateFirstCardDealt RoundState = "first-card-dealt"

	// RoundStatePendingAceDecision means the first card has been dealt, it's an ace, and the player needs to pick high/low
	RoundStatePendingAceDecision RoundState = "pending-ace-decision"

	// RoundStatePendingBet means the last card has been dealt and we are waiting for participant to place bet
	RoundStatePendingBet RoundState = "pending-bet"

	// RoundStateBetPlaced means a bet has been successfully placed
	RoundStateBetPlaced RoundState = "bet-placed"

	// RoundStateGameOver means the game ended and there's still at least one more game to be played
	RoundStateGameOver RoundState = "game-over"

	// RoundStateRoundOver means all games have finished
	RoundStateRoundOver RoundState = "round-over"

	// RoundStateComplete means there are no more rounds to be played
	RoundStateComplete RoundState = "complete"

	// RoundStateWaiting means we are waiting until a time passes before proceeding
	RoundStateWaiting RoundState = "waiting"
)

// NewRound returns a new Round object
func NewRound(playerID int64, d *deck.Deck, startingPot int) *Round {
	return &Round{
		PlayerID: playerID,
		Games:    []*SingleGame{newSingleGame()},
		State:    RoundStateStart,
		Pot:      startingPot,

		activeGameIndex: 0,
		deck:            d,
	}
}

// DealCard deals a card in Acey Deucey
func (r *Round) DealCard() error {
	if _, err := r.activeGame(); err != nil {
		return err
	}

	card, err := r.drawCard()
	if err != nil {
		return err
	}

	switch r.State {
	case RoundStateStart:
		r.sendLogMessage("Left card dealt", card, 0)
		r.dealFirstCard(card)
		return nil
	case RoundStateFirstCardDealt:
		if err := r.dealLastCard(card); err != nil {
			r.deck.UndoDraw(card)
			return err
		}

		// if we are back to the same state, it's because there's a free game
		if r.State == RoundStateFirstCardDealt {
			r.sendLogMessage("Bonus game", card, 0)
		} else {
			r.sendLogMessage("Right card dealt", card, 0)
		}

		return nil
	case RoundStateBetPlaced:
		r.sendLogMessage("Middle card dealt", card, 0)
		r.dealMiddleCard(card)
		return nil
	}

	r.deck.UndoDraw(card)
	return fmt.Errorf("cannot deal card from state: %s", r.State)
}

// SetBet will set an active bet
func (r *Round) SetBet(bet int, isHalfPotBet bool) error {
	game, err := r.activeGame()
	if err != nil {
		return err
	}

	if r.State != RoundStatePendingBet {
		return fmt.Errorf("cannot place a bet from state: %s", r.State)
	}

	if bet == 0 {
		return errors.New("bet must be at least ${25}")
	}

	if bet%25 > 0 {
		return errors.New("bet must be in increments of ${25}")
	}

	if maxBet := r.getMaxBet(); bet > maxBet {
		return fmt.Errorf("bet of ${%d} exceeds the max bet of ${%d}", bet, maxBet)
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

	if game.Bet.HalfPot {
		r.sendLogMessage(fmt.Sprintf("{} bet ${%d} for half-pot", game.Bet.Amount), nil, r.PlayerID)
	} else {
		r.sendLogMessage(fmt.Sprintf("{} bet ${%d}", game.Bet.Amount), nil, r.PlayerID)
	}

	r.State = RoundStateBetPlaced
	return nil
}

// dealFirstCard must only be called from DealCard()
func (r *Round) dealFirstCard(card *deck.Card) {
	game := r.Games[r.activeGameIndex]
	game.FirstCard = card
	if card.Rank == deck.Ace {
		card.SetBit(aceStateUndecided)
		r.State = RoundStatePendingAceDecision
		return
	}

	r.State = RoundStateFirstCardDealt
}

// dealLastCard must only be called from DealCard()
func (r *Round) dealLastCard(card *deck.Card) error {
	game, err := r.activeGame()
	if err != nil {
		return err
	}

	if card := game.FirstCard; card.Rank == deck.Ace {
		if !card.IsBitSet(aceStateLow) && !card.IsBitSet(aceStateHigh) {
			panic("bit not properly set on first ace")
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
		r.finalizeGame(game, SingleGameResultFreeGame, 0)
		return nil
	}

	r.State = RoundStatePendingBet
	return nil
}

// finalizeGame updates balances and sets the state
func (r *Round) finalizeGame(g *SingleGame, result SingleGameResult, adjustment int) {
	g.Adjustment = adjustment
	g.Result = result
	r.Pot -= adjustment

	switch result {
	case SingleGameResultFreeGame:
		r.sendLogMessage("{} received a free game", nil, r.PlayerID)
	case SingleGameResultPost:
		r.sendLogMessage(fmt.Sprintf("{} posted and lost ${%d}", adjustment), nil, r.PlayerID)
	default:
		r.sendLogMessage(fmt.Sprintf("{} %s ${%d}", result, adjustment), nil, r.PlayerID)
	}

	// if this is the last game or the Pot is empty, end it
	if r.activeGameIndex+1 == len(r.Games) || r.Pot == 0 {
		r.setNextState(RoundStateRoundOver, time.Second*1)
	} else {
		r.setNextState(RoundStateGameOver, time.Second*1)
	}
}

// dealMiddleCard must only be called from DealCard()
func (r *Round) dealMiddleCard(card *deck.Card) {
	game := r.Games[r.activeGameIndex]
	game.MiddleCard = card

	firstCardRank := game.firstCardRank()

	if card.Rank == firstCardRank || card.Rank == game.LastCard.Rank {
		r.finalizeGame(game, SingleGameResultPost, -2*game.Bet.Amount)
		return
	}

	lowCard, highCard := firstCardRank, game.LastCard.Rank
	if firstCardRank > game.LastCard.Rank {
		// swap them around
		lowCard, highCard = highCard, lowCard
	}

	if card.Rank > lowCard && card.Rank < highCard {
		if game.Bet.HalfPot {
			r.finalizeGame(game, SingleGameResultWon, r.getHalfPot())
		} else {
			r.finalizeGame(game, SingleGameResultWon, game.Bet.Amount)
		}

		return
	}

	r.finalizeGame(game, SingleGameResultLost, -1*game.Bet.Amount)
}

// getHalfPot returns half of the pot, rounded down to the nearest 25
func (r *Round) getHalfPot() int {
	halfPot := r.Pot / 2
	halfPot -= halfPot % 25

	return halfPot
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

	if r.Pot < betTheGapAmount*2 {
		return false
	}

	return math.Abs(float64(game.firstCardRank()-game.LastCard.Rank)) == 2
}

func (r *Round) nextGame() error {
	if r.State != RoundStateGameOver {
		return fmt.Errorf("invalid state to move to next game: %s", r.State)
	}

	r.activeGameIndex++
	if r.Games[r.activeGameIndex].FirstCard.Rank == deck.Ace {
		r.State = RoundStatePendingAceDecision
	} else {
		r.State = RoundStateFirstCardDealt
	}

	return nil
}

// SetAce will set whether the first ace is low or high
func (r *Round) SetAce(highAce bool) error {
	game, err := r.activeGame()
	if err != nil {
		return err
	}

	if r.State != RoundStatePendingAceDecision {
		return fmt.Errorf("cannot choose ace low/high from state: %s", r.State)
	}

	card := game.FirstCard
	if card.Rank != deck.Ace {
		panic(fmt.Sprintf("first card is %s, but the state is %s", card.String(), r.State))
	}

	card.UnsetAllBits()
	bit := aceStateLow
	chose := "low"
	if highAce {
		bit = aceStateHigh
		chose = "high"
	}

	r.sendLogMessage(fmt.Sprintf("{} chose ace %s", chose), nil, r.PlayerID)
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

func (r *Round) setNextState(state RoundState, after time.Duration) {
	r.nextAction = &nextAction{
		Time:      time.Now().Add(after),
		NextState: state,
	}

	r.State = RoundStateWaiting
}

func (r *Round) checkWaiting() {
	if r.nextAction != nil {
		if time.Now().After(r.nextAction.Time) {
			r.State = r.nextAction.NextState
			r.nextAction = nil
		}
	}
}

func (r *Round) sendLogMessage(message string, card *deck.Card, playerID int64) {
	if r.logChan != nil {
		var playerIDs []int64
		if playerID > 0 {
			playerIDs = []int64{playerID}
		}

		var cards []*deck.Card
		if card != nil {
			cards = []*deck.Card{card}
		}

		r.logChan <- []*playable.LogMessage{
			{
				UUID:      uuid.New().String(),
				PlayerIDs: playerIDs,
				Cards:     cards,
				Message:   message,
			},
		}
	}
}

func (r *Round) getMaxBet() int {
	if r.Pot <= 0 {
		return 0
	}

	// if HalfPotMax is false, the pot is the max bet
	if !r.HalfPotMax {
		return r.Pot
	}

	halfPot := r.getHalfPot()
	if halfPot < 50 {
		return 25
	}

	return halfPot
}
