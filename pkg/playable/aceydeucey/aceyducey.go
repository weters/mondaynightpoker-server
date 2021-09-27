package aceydeucey

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"strings"
	"time"
)

// betTheAmount is the standard bet for a half-pot bet the gap
// Should we make this configurable? Maybe double the ante?
const betTheGapAmount = 50

// Game is a game of Acey Deucey
type Game struct {
	options             Options
	orderedParticipants []*Participant
	participants        map[int64]*Participant
	deck                *deck.Deck
	logChan             chan []*playable.LogMessage
	turnIndex           int
	logger              logrus.FieldLogger

	pot    int
	rounds []*Round
}

// Interval is how long we should wait before updating the game state
func (g *Game) Interval() time.Duration {
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
	d.Shuffle()

	a := &Game{
		options:             options,
		orderedParticipants: orderedParticipants,
		participants:        idToParticipant,
		deck:                d,
		logChan:             make(chan []*playable.LogMessage, 256),
		turnIndex:           0,
		pot:                 len(playerIDs) * options.Ante,
		logger:              logger,
	}

	a.newRound()
	return a, nil
}

// NameFromOptions returns the name for the options
func NameFromOptions(opts Options) string {
	options := make([]string, 0, 2)
	switch opts.GameType {
	case GameTypeContinuousShoe:
		options = append(options, "Continuous Shoe")
	case GameTypeChaos:
		options = append(options, "Chaos")
	}

	if opts.AllowPass {
		options = append(options, "With Passing")
	}

	const name = "Acey Deucey"
	if len(options) > 0 {
		return fmt.Sprintf("%s (%s)", name, strings.Join(options, " and "))
	}

	return name
}

// Name returns the name of the game
func (g *Game) Name() string {
	return NameFromOptions(g.options)
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

	round := g.getCurrentRound()

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
		if err := round.SetPass(); err != nil {
			return nil, false, err
		}
		return playable.OK(), true, nil
	case ActionBetTheGap:
		if err := round.SetBet(betTheGapAmount, true); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	case ActionBet:
		amount, _ := message.AdditionalData.GetInt("amount")

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
	if g.getCurrentRound().State != RoundStateComplete {
		return nil, false
	}

	adjustments := make(map[int64]int)
	for _, p := range g.participants {
		adjustments[p.PlayerID] = p.Balance
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: adjustments,
		Log:                g.rounds,
	}, true
}

// LogChan should return a channel that a game will send log messages to
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}

func (g *Game) getCurrentTurn() *Participant {
	id := g.orderedParticipants[g.turnIndex].PlayerID
	participant, ok := g.participants[id]
	if !ok {
		panic(fmt.Sprintf("inconsistent state found, player ID %d not in participants", id))
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

// newRound starts a new round
// NOTE: do not call this method until the correct participant is lined up
func (g *Game) newRound() {
	if g.options.GameType == GameTypeContinuousShoe {
		g.deck.Shuffle()
	}

	turn := g.getCurrentTurn()
	r := NewRound(g.options, turn.PlayerID, g.deck, g.pot)
	r.logChan = g.logChan
	g.rounds = append(g.rounds, r)

	// limit the bet to half the pot if not all participants went yet
	r.HalfPotMax = len(g.rounds) <= len(g.participants)
}

func (g *Game) endRound() error {
	participant := g.getCurrentTurn()
	if participant == nil {
		return errors.New("no activate participant")
	}

	currentRound := g.getCurrentRound()
	g.pot = currentRound.Pot
	participant.Balance += currentRound.ParticipantAdjustments()
	if g.pot > 0 {
		g.nextTurn()
		g.newRound()

		return nil
	}

	currentRound.setNextState(RoundStateComplete, time.Second*2)
	return nil
}

// Tick is periodically called and will try to advance the game state
func (g *Game) Tick() (didUpdate bool, err error) {
	currentRound := g.getCurrentRound()
	switch currentRound.State {
	case RoundStateStart:
		fallthrough
	case RoundStateBetPlaced:
		fallthrough
	case RoundStateFirstCardDealt:
		if err := currentRound.DealCard(); err != nil {
			return false, err
		}

		return true, nil
	case RoundStatePassed:
		currentRound.PassRound()
		return true, nil
	case RoundStateGameOver:
		if err := currentRound.nextGame(); err != nil {
			return false, err
		}

		return true, nil
	case RoundStateRoundOver:
		if err := g.endRound(); err != nil {
			return false, err
		}

		return true, nil
	case RoundStateWaiting:
		currentRound.checkWaiting()
	}

	return false, nil
}

func (g *Game) getCurrentRound() *Round {
	return g.rounds[len(g.rounds)-1]
}
