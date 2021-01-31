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

// AceyDeucey is a game of Acey Deucey
type AceyDeucey struct {
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
func (a *AceyDeucey) Delay() time.Duration {
	return time.Second
}

// NewGame returns a new game
func NewGame(logger logrus.FieldLogger, playerIDs []int64, options Options) (*AceyDeucey, error) {
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

	a := &AceyDeucey{
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
func (a *AceyDeucey) Name() string {
	return "Acey Deucey"
}

// Key returns a unique key
func (a *AceyDeucey) Key() string {
	return "acey-deucey"
}

// Action performs with a message
// If playerResponse is not null, that's the response sent directly to the client
// If updateState is true, it will trigger a state update for all connected clients
func (a *AceyDeucey) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	actions := a.getActionsForParticipant(playerID)
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

	round := a.currentRound

	switch action {
	case ActionPickAceLow:
		if err := round.setAce(false); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	case ActionPickAceHigh:
		if err := round.setAce(true); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	case ActionPass:
		panic("implement me")
	case ActionBetTheGap:
		amount := a.options.Ante * 2
		if err := round.setBet(amount, true); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	case ActionBet:
		amount, _ := message.AdditionalData.GetInt("amount")

		if amount%25 > 0 || amount == 0 {
			return nil, false, errors.New("bet must be in multiples of 25 cents")
		}

		if err := round.setBet(amount, false); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	}

	panic("implement me")
}

// GetPlayerState returns the current state of the game for the player
func (a *AceyDeucey) GetPlayerState(playerID int64) (*playable.Response, error) {
	gameState := a.getParticipantState(playerID)
	return &playable.Response{
		Key:   "game",
		Value: a.Key(),
		Data:  gameState,
	}, nil
}

// GetEndOfGameDetails returns the details after a game is over
// If the game is still in progress, nil will be returned and the second param will be false
func (a *AceyDeucey) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	if a.currentRound.State != RoundStateComplete {
		return nil, false
	}

	adjustments := make(map[int64]int)
	for _, p := range a.participants {
		adjustments[p.PlayerID] = p.Balance
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: adjustments,
		Log:                nil,
	}, true
}

// LogChan should return a channel that a game will send log messages to
func (a *AceyDeucey) LogChan() <-chan []*playable.LogMessage {
	return nil
}

func (a *AceyDeucey) getCurrentTurn() *Participant {
	id := a.orderedParticipants[a.turnIndex].PlayerID
	participant, ok := a.participants[id]
	if !ok {
		return nil
	}

	return participant
}

func (a *AceyDeucey) nextTurn() {
	a.turnIndex++
	a.turnIndex = a.turnIndex % len(a.orderedParticipants)
}

// isGameOver returns true if the pot is empty
func (a *AceyDeucey) isGameOver() bool {
	return a.pot == 0
}

func (a *AceyDeucey) newRound() {
	a.currentRound = NewRound(a.deck, a.pot)
}

func (a *AceyDeucey) endRound() error {
	participant := a.getCurrentTurn()
	if participant == nil {
		return errors.New("no activate participant")
	}

	a.pot = a.currentRound.Pot
	participant.Balance += a.currentRound.ParticipantAdjustments()
	if a.pot > 0 {
		a.nextTurn()
		a.newRound()

		return nil
	}

	a.currentRound.setNextState(RoundStateComplete, time.Second*2)
	return nil
}

// Tick is called when the game state should advance
func (a *AceyDeucey) Tick() (bool, error) {
	switch a.currentRound.State {
	case RoundStateStart:
		fallthrough
	case RoundStateBetPlaced:
		fallthrough
	case RoundStateFirstCardDealt:
		logrus.Info(a.currentRound.State)
		if err := a.currentRound.DealCard(); err != nil {
			return false, err
		}

		return true, nil
	case RoundStateGameOver:
		logrus.Info(a.currentRound.State)
		if err := a.currentRound.nextGame(); err != nil {
			return false, err
		}

		return true, nil
	case RoundStateRoundOver:
		logrus.Info(a.currentRound.State)
		if err := a.endRound(); err != nil {
			return false, err
		}

		return true, nil
	case RoundStateWaiting:
		a.currentRound.checkWaiting()
	}

	return false, nil
}
