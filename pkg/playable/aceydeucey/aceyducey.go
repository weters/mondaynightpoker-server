package aceydeucey

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"
)

var seed = int64(0)

// Game is a game of Acey Deucey
type Game struct {
	options             Options
	orderedParticipants []*Participant
	participants        map[int64]*Participant
	deck                *deck.Deck
	logChan             chan []playable.LogMessage
	turnIndex           int

	pot          int
	currentRound *Round
}

// Delay is how long we should wait before updating the game state
func (g *Game) Delay() time.Duration {
	return time.Second
}

// NewGame returns a new game
func NewGame(logger logrus.FieldLogger, playerIDs []int64, options Options) (*Game, error) {
	if len(playerIDs) < 2 {
		return nil, errors.New("game requires at least two players")
	}

	if options.Ante <= 0 {
		return nil, errors.New("ante must be > 0")
	}

	orderedParticipants := make([]*Participant, len(playerIDs))
	idToParticipant := make(map[int64]*Participant, len(playerIDs))
	for i, pid := range playerIDs {
		p := NewParticipant(pid, options.Ante)
		idToParticipant[pid] = p
		orderedParticipants[i] = p
	}

	if len(playerIDs) != len(idToParticipant) {
		return nil, errors.New("duplicate players detected")
	}

	d := deck.New()
	d.Shuffle(seed)

	a := &Game{
		options:             options,
		orderedParticipants: orderedParticipants,
		participants:        idToParticipant,
		deck:                d,
		logChan:             make(chan []playable.LogMessage, 256),
		turnIndex:           0,
		pot:                 len(playerIDs) * options.Ante,
	}

	a.newRound()
	return a, nil
}

// Name returns the name of the game
func (g *Game) Name() string {
	return "Acey Deucey"
}

// Key returns a unique key
func (g *Game) Key() string {
	return "acey-deucey"
}

// Action performs with a message
// If playerResponse is not null, that's the response sent directly to the client
// If updateState is true, it will trigger a state update for all connected clients
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	actions := g.getActionsForParticipant(playerID)
	action, err := ActionFromString(message.Subject)
	if err != nil {
		return nil, false, err
	}

	isValidAction := false
	for _, validAction := range actions {
		if action == validAction {
			isValidAction = true
			break
		}
	}

	if !isValidAction {
		return nil, false, fmt.Errorf("you cannot perform the action: %s", action.String())
	}

	round := g.currentRound

	switch action {
	case ActionPickAceLow:
		if err := round.SetAce(false); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	case ActionPickAceHigh:
		if err := round.SetAce(true); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	case ActionPass:
		panic("implement me")
	case ActionBetTheGap:
		amount := g.options.Ante * 2
		if err := round.SetBet(amount, true); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	case ActionBet:
		amount, _ := message.AdditionalData.GetInt("amount")

		if amount%25 > 0 || amount == 0 {
			return nil, false, errors.New("bet must be in multiples of 25 cents")
		}

		if err := round.SetBet(amount, false); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	}

	panic("implement me")
}

// GetPlayerState returns the current state of the game for the player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	gameState := g.getParticipantState(playerID)
	return &playable.Response{
		Key:   "game",
		Value: g.Key(),
		Data:  gameState,
	}, nil
}

// GetEndOfGameDetails returns the details after a game is over
// If the game is still in progress, nil will be returned and the second param will be false
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	if g.currentRound.State != RoundStateComplete {
		return nil, false
	}

	adjustments := make(map[int64]int)
	for _, p := range g.participants {
		adjustments[p.PlayerID] = p.Balance
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: adjustments,
		Log:                nil,
	}, true
}

// LogChan should return a channel that a game will send log messages to
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return nil
}

func (g *Game) getCurrentTurn() *Participant {
	id := g.orderedParticipants[g.turnIndex].PlayerID
	participant, ok := g.participants[id]
	if !ok {
		return nil
	}

	return participant
}

func (g *Game) nextTurn() {
	g.turnIndex++
	g.turnIndex = g.turnIndex % len(g.orderedParticipants)
}

// isGameOver returns true if the pot is empty
func (g *Game) isGameOver() bool {
	return g.pot == 0
}

func (g *Game) newRound() {
	g.currentRound = NewRound(g.deck, g.pot)
}

func (g *Game) endRound() error {
	participant := g.getCurrentTurn()
	if participant == nil {
		return errors.New("no activate participant")
	}

	g.pot = g.currentRound.Pot
	participant.Balance += g.currentRound.ParticipantAdjustments()
	if g.pot > 0 {
		g.nextTurn()
		g.newRound()

		return nil
	}

	g.currentRound.setNextState(RoundStateComplete, time.Second*2)
	return nil
}

// Tick is called when the game state should advance
func (g *Game) Tick() (bool, error) {
	switch g.currentRound.State {
	case RoundStateStart:
		fallthrough
	case RoundStateBetPlaced:
		fallthrough
	case RoundStateFirstCardDealt:
		logrus.Info(g.currentRound.State)
		if err := g.currentRound.DealCard(); err != nil {
			return false, err
		}

		return true, nil
	case RoundStateGameOver:
		logrus.Info(g.currentRound.State)
		if err := g.currentRound.nextGame(); err != nil {
			return false, err
		}

		return true, nil
	case RoundStateRoundOver:
		logrus.Info(g.currentRound.State)
		if err := g.endRound(); err != nil {
			return false, err
		}

		return true, nil
	case RoundStateWaiting:
		g.currentRound.checkWaiting()
	}

	return false, nil
}
