package guts

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Phase represents the current phase of the game
type Phase int

const (
	// PhaseDealing is when cards are being dealt
	PhaseDealing Phase = iota
	// PhaseDeclaration is when players decide in/out
	PhaseDeclaration
	// PhaseShowdown is when hands are revealed and compared
	PhaseShowdown
	// PhaseRoundEnd is the end of a round before the next
	PhaseRoundEnd
	// PhaseGameOver is when the game has ended
	PhaseGameOver
)

// ShowdownResult contains the results of a showdown
type ShowdownResult struct {
	Winners      []*Participant
	Losers       []*Participant
	PlayersIn    []*Participant
	WinningHand  HandResult
	PotWon       int
	PenaltyPaid  int
	NextPot      int
	AllFolded    bool
	SingleWinner bool
}

// Game is a game of 2-card guts
type Game struct {
	options         Options
	deck            *deck.Deck
	participants    []*Participant
	idToParticipant map[int64]*Participant

	pot         int
	phase       Phase
	roundNumber int

	// Simultaneous declaration tracking
	pendingDecisions map[int64]bool // Who hasn't decided yet
	decisions        map[int64]bool // true=In, false=Out

	showdownResult *ShowdownResult

	done bool

	logger  logrus.FieldLogger
	logChan chan []*playable.LogMessage

	pendingDealerAction *pendingDealerAction
}

// Interval determines how often Tick() should be called
func (g *Game) Interval() time.Duration {
	return time.Second
}

// Tick will check the state of the game and possibly move the state along
func (g *Game) Tick() (bool, error) {
	if g.done {
		return false, nil
	}

	if g.pendingDealerAction != nil {
		if time.Now().After(g.pendingDealerAction.ExecuteAfter) {
			action := g.pendingDealerAction.Action
			// Clear BEFORE executing so actions can schedule new ones
			g.pendingDealerAction = nil

			switch action {
			case dealerActionShowdown:
				g.calculateShowdown()
			case dealerActionNextRound:
				if err := g.nextRound(); err != nil {
					logrus.WithError(err).Error("could not go to the next round")
				}
			case dealerActionEndGame:
				g.done = true
			default:
				panic(fmt.Sprintf("unknown dealer action: %d", action))
			}

			return true, nil
		}

		return false, nil
	}

	return false, nil
}

// Name returns "guts"
func (g *Game) Name() string {
	return "guts"
}

// LogChan returns a channel for sending log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}

// Action performs an action
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	if g.phase == PhaseGameOver {
		return nil, false, ErrGameIsOver
	}

	_, ok := g.idToParticipant[playerID]
	if !ok {
		return nil, false, ErrPlayerNotFound
	}

	switch message.Action {
	case "decide":
		goIn, ok := message.AdditionalData.GetBool("in")
		if !ok {
			return nil, false, errors.New("missing 'in' parameter")
		}

		if err := g.submitDecision(playerID, goIn); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	default:
		return nil, false, fmt.Errorf("unknown action: %s", message.Action)
	}
}

// GetEndOfGameDetails returns details at the end of the game
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	if !g.done {
		return nil, false
	}

	adjustments := make(map[int64]int)
	for _, p := range g.participants {
		adjustments[p.PlayerID] = p.balance
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: adjustments,
		Log:                g.showdownResult,
	}, true
}

// NewGame returns a new guts game
func NewGame(logger logrus.FieldLogger, playerIDs []int64, opts Options) (*Game, error) {
	if len(playerIDs) < 2 || len(playerIDs) > 10 {
		return nil, PlayerCountError{
			Min: 2,
			Max: 10,
			Got: len(playerIDs),
		}
	}

	participants := make([]*Participant, len(playerIDs))
	idToParticipant := make(map[int64]*Participant)

	for i, pid := range playerIDs {
		p := NewParticipant(pid)
		participants[i] = p
		idToParticipant[pid] = p
	}

	d := deck.New()
	d.Shuffle()

	pot := 0
	messages := make([]*playable.LogMessage, 0)

	for _, p := range participants {
		pot += opts.Ante
		p.balance -= opts.Ante
		messages = append(messages, newLogMessage(p.PlayerID, "{} paid the ${%d} ante", opts.Ante))
	}

	g := &Game{
		options:          opts,
		deck:             d,
		participants:     participants,
		idToParticipant:  idToParticipant,
		pot:              pot,
		phase:            PhaseDealing,
		roundNumber:      1,
		pendingDecisions: make(map[int64]bool),
		decisions:        make(map[int64]bool),
		logger:           logger,
		logChan:          make(chan []*playable.LogMessage, 256),
	}

	messages = append(messages, newLogMessage(0, "New game of %s started with a pot of ${%d}", NameFromOptions(opts), pot))
	g.sendLogMessages(messages...)

	return g, nil
}

// Deal will deal cards to each participant
func (g *Game) Deal() error {
	if len(g.participants) < 2 {
		return ErrNotEnoughPlayers
	}

	// Clear hands and reset for new round
	for _, p := range g.participants {
		p.ClearHand()
	}

	// Deal cards to each player
	cardCount := g.options.CardCount
	if cardCount < 2 || cardCount > 3 {
		cardCount = 2
	}
	for i := 0; i < cardCount; i++ {
		for _, p := range g.participants {
			card, err := g.deck.Draw()
			if err != nil {
				return err
			}
			p.AddCard(card)
		}
	}

	// Initialize pending decisions
	g.pendingDecisions = make(map[int64]bool)
	g.decisions = make(map[int64]bool)
	for _, p := range g.participants {
		g.pendingDecisions[p.PlayerID] = true
	}

	g.phase = PhaseDeclaration
	g.sendLogMessages(newLogMessage(0, "Round %d: Cards dealt, declare In or Out", g.roundNumber))

	return nil
}

// submitDecision records a player's in/out decision
func (g *Game) submitDecision(playerID int64, goIn bool) error {
	if g.phase != PhaseDeclaration {
		return ErrNotInDeclarationPhase
	}

	if !g.pendingDecisions[playerID] {
		return ErrAlreadyDecided
	}

	g.decisions[playerID] = goIn
	delete(g.pendingDecisions, playerID)

	// Log that player has decided (without revealing the decision)
	g.sendLogMessages(newLogMessage(playerID, "{} has decided"))

	// Only reveal when ALL have decided
	if len(g.pendingDecisions) == 0 {
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionShowdown,
			ExecuteAfter: time.Now().Add(time.Second),
		}
	}

	return nil
}

// calculateShowdown determines winners and losers
func (g *Game) calculateShowdown() {
	g.phase = PhaseShowdown

	// Reveal all decisions now that everyone has decided
	for _, p := range g.participants {
		decision := "Out"
		if g.decisions[p.PlayerID] {
			decision = "In"
		}
		g.sendLogMessages(newLogMessage(p.PlayerID, "{} was %s", decision))
	}

	// Find players who went in
	playersIn := make([]*Participant, 0)
	for _, p := range g.participants {
		if g.decisions[p.PlayerID] {
			playersIn = append(playersIn, p)
		}
	}

	result := &ShowdownResult{
		PlayersIn: playersIn,
	}

	// Case 1: No one goes in - everyone re-antes
	if len(playersIn) == 0 {
		result.AllFolded = true
		g.showdownResult = result
		g.sendLogMessages(newLogMessage(0, "No one went in! Everyone re-antes."))

		// Schedule next round
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionNextRound,
			ExecuteAfter: time.Now().Add(time.Second * 5),
		}
		return
	}

	// Case 2: Only one person goes in - they win the pot outright
	if len(playersIn) == 1 {
		winner := playersIn[0]
		winner.balance += g.pot
		result.Winners = []*Participant{winner}
		result.WinningHand = AnalyzeHand(winner.hand)
		result.PotWon = g.pot
		result.SingleWinner = true
		g.showdownResult = result

		g.sendLogMessages(newLogMessage(winner.PlayerID, "{} wins ${%d} (only one in)", g.pot))

		// Game ends
		g.phase = PhaseGameOver
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionEndGame,
			ExecuteAfter: time.Now().Add(time.Second * 2),
		}
		return
	}

	// Case 3: Multiple players go in - compare hands
	// Find the best hand(s)
	var bestStrength int
	winners := make([]*Participant, 0)

	for _, p := range playersIn {
		handResult := AnalyzeHand(p.hand)
		if handResult.Strength > bestStrength {
			bestStrength = handResult.Strength
			winners = []*Participant{p}
		} else if handResult.Strength == bestStrength {
			winners = append(winners, p)
		}
	}

	// Determine losers (in players who didn't win)
	losers := make([]*Participant, 0)
	for _, p := range playersIn {
		isWinner := false
		for _, w := range winners {
			if w.PlayerID == p.PlayerID {
				isWinner = true
				break
			}
		}
		if !isWinner {
			losers = append(losers, p)
		}
	}

	result.Winners = winners
	result.Losers = losers
	result.WinningHand = AnalyzeHand(winners[0].hand)
	result.PotWon = g.pot

	// Distribute pot to winners
	winPerPerson := g.pot / len(winners)
	remainder := g.pot % len(winners)
	for i, w := range winners {
		winAmount := winPerPerson
		if i < remainder {
			winAmount++ // Distribute remainder
		}
		w.balance += winAmount
	}

	// Calculate penalty (capped at maxOwed)
	penalty := g.calculatePenalty()
	result.PenaltyPaid = penalty

	// Losers pay penalty into next pot
	nextPot := 0
	for _, loser := range losers {
		loser.balance -= penalty
		nextPot += penalty
	}
	result.NextPot = nextPot

	g.showdownResult = result

	// Log results
	if len(winners) == 1 {
		g.sendLogMessages(newLogMessage(winners[0].PlayerID, "{} wins ${%d} with %s",
			g.pot, HandTypeName(result.WinningHand.Type)))
	} else {
		playerIDs := make([]int64, len(winners))
		for i, w := range winners {
			playerIDs[i] = w.PlayerID
		}
		g.sendLogMessages(newLogMessageWithPlayers(playerIDs, "{} split the pot of ${%d}", g.pot))
	}

	for _, loser := range losers {
		g.sendLogMessages(newLogMessage(loser.PlayerID, "{} pays penalty of ${%d}", penalty))
	}

	// If there are losers who paid penalties, continue the game
	if nextPot > 0 {
		g.pot = nextPot
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionNextRound,
			ExecuteAfter: time.Now().Add(time.Second * 5),
		}
	} else {
		// No penalties paid (everyone who was in won), game ends
		g.phase = PhaseGameOver
		g.sendLogMessages(newLogMessage(0, "The game ends"))
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionEndGame,
			ExecuteAfter: time.Now().Add(time.Second * 2),
		}
	}
}

// calculatePenalty returns the penalty amount (pot capped at maxOwed)
func (g *Game) calculatePenalty() int {
	if g.pot > g.options.MaxOwed {
		return g.options.MaxOwed
	}
	return g.pot
}

// nextRound starts a new round
func (g *Game) nextRound() error {
	// If everyone folded, re-ante
	if g.showdownResult != nil && g.showdownResult.AllFolded {
		for _, p := range g.participants {
			p.balance -= g.options.Ante
			g.pot += g.options.Ante
		}
		g.sendLogMessages(newLogMessage(0, "Everyone re-anted. Pot is now ${%d}", g.pot))
	}

	g.roundNumber++
	g.showdownResult = nil

	// Reshuffle deck
	g.deck = deck.New()
	g.deck.Shuffle()

	// Deal new round
	return g.Deal()
}

func (g *Game) sendLogMessages(msg ...*playable.LogMessage) {
	if g.logChan != nil {
		g.logChan <- msg
	}
}

func newLogMessage(playerID int64, format string, a ...interface{}) *playable.LogMessage {
	return &playable.LogMessage{
		UUID:      uuid.New().String(),
		PlayerIDs: []int64{playerID},
		Message:   fmt.Sprintf(format, a...),
		Time:      time.Now(),
	}
}

func newLogMessageWithPlayers(playerIDs []int64, format string, a ...interface{}) *playable.LogMessage {
	return &playable.LogMessage{
		UUID:      uuid.New().String(),
		PlayerIDs: playerIDs,
		Message:   fmt.Sprintf(format, a...),
		Time:      time.Now(),
	}
}

// NameFromOptions returns the name of the game based on options
func NameFromOptions(opts Options) string {
	if opts.CardCount == 3 {
		return "3-Card Guts"
	}
	return "2-Card Guts"
}
