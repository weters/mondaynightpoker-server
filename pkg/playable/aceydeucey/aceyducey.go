package aceydeucey

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
)

var seed = int64(0)

// AceyDeucey is a game of Acey Ducey
type AceyDeucey struct {
	options      Options
	playerIDs    []int64
	participants map[int64]*Participant
	deck         *deck.Deck
	logChan      chan []playable.LogMessage
	turnIndex    int

	pot          int
	currentRound *round
}

// NewGame returns a new game
func NewGame(logger logrus.FieldLogger, playerIDs []int64, options Options) (*AceyDeucey, error) {
	if len(playerIDs) < 2 {
		return nil, errors.New("game requires at least two players")
	}

	if options.Ante <= 0 {
		return nil, errors.New("ante must be > 0")
	}

	idToParticipant := make(map[int64]*Participant, len(playerIDs))
	for _, pid := range playerIDs {
		idToParticipant[pid] = NewParticipant(pid, options.Ante)
	}

	if len(playerIDs) != len(idToParticipant) {
		return nil, errors.New("duplicate players detected")
	}

	localPlayerIds := make([]int64, len(playerIDs))
	copy(localPlayerIds, playerIDs)

	d := deck.New()
	d.Shuffle(seed)

	a := &AceyDeucey{
		options:      options,
		playerIDs:    localPlayerIds,
		participants: idToParticipant,
		deck:         d,
		logChan:      make(chan []playable.LogMessage, 256),
		turnIndex:    0,
		pot:          len(playerIDs) * options.Ante,
	}

	if err := a.newRound(); err != nil {
		return nil, err
	}

	return a, nil
}

// Name returns the name of the game
func (a *AceyDeucey) Name() string {
	return "Acey Ducey"
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

	switch action {
	case ActionPickAceLow:
	case ActionPickAceHigh:
	case ActionPass:
	case ActionBet:
	case ActionContinue:
	}

	panic("implement me")
}

// GetPlayerState returns the current state of the game for the player
func (a *AceyDeucey) GetPlayerState(playerID int64) (*playable.Response, error) {
	panic("implement me")
}

// GetEndOfGameDetails returns the details after a game is over
// If the game is still in progress, nil will be returned and the second param will be false
func (a *AceyDeucey) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	panic("implement me")
}

// LogChan should return a channel that a game will send log messages to
func (a *AceyDeucey) LogChan() <-chan []*playable.LogMessage {
	panic("implement me")
}

func (a *AceyDeucey) getCurrentTurn() *Participant {
	id := a.playerIDs[a.turnIndex]
	participant, ok := a.participants[id]
	if !ok {
		return nil
	}

	return participant
}

func (a *AceyDeucey) nextTurn() {
	a.turnIndex++
	a.turnIndex = a.turnIndex % len(a.playerIDs)
}

// isGameOver returns true if the pot is empty
func (a *AceyDeucey) isGameOver() bool {
	return a.pot == 0
}

func (a *AceyDeucey) newRound() error {
	a.currentRound = newRound()
	return a.advance()
}

// advance will
func (a *AceyDeucey) advance() error {
	if a.isGameOver() {
		return fmt.Errorf("%s is over", a.Name())
	}
	return nil
}
