package texasholdem

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"
)

const anteMin = 0
const anteMax = 50
const lowerLimitMax = 100

var errBettingRoundIsOver = errors.New("betting round is over")

type lastAction struct {
	Action   Action `json:"action"`
	PlayerID int64  `json:"playerId"`
}

// Game is a game of Limit Texas Hold'em
type Game struct {
	options            Options
	deck               *deck.Deck
	participants       map[int64]*Participant
	participantOrder   []int64
	dealerState        DealerState
	pendingDealerState *pendingDealerState
	decisionIndex      int
	decisionStart      int
	pot                int
	currentBet         int
	lastAction         *lastAction
	community          deck.Hand
	logChan            chan []*playable.LogMessage

	// if true, GetEndOfGameDetails() returns
	finished bool
}

// Options configures how Texas Hold'em is played
type Options struct {
	Ante       int
	LowerLimit int
	UpperLimit int
}

// DefaultOptions returns the default options for Texas Hold'em
func DefaultOptions() Options {
	return Options{
		Ante:       25,
		LowerLimit: 100,
		UpperLimit: 200,
	}
}

// NewGame returns a new game of Texas Hold'em
func NewGame(logger logrus.FieldLogger, playerIDs []int64, opts Options) (*Game, error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	opts.UpperLimit = opts.LowerLimit * 2

	if len(playerIDs) < 2 {
		return nil, errors.New("there must be at least two players")
	}

	d := deck.New()
	d.Shuffle()

	smallBlind := (opts.LowerLimit / 50) * 25
	if smallBlind == 0 {
		smallBlind = opts.LowerLimit
	}

	participants := make(map[int64]*Participant)
	participantOrder := make([]int64, len(playerIDs))
	copy(participantOrder, playerIDs)

	logs := make([]*playable.LogMessage, 0)
	logs = append(logs, playable.SimpleLogMessage(0, "started a new game of %s", NameFromOptions(opts)))
	blindLogs := make([]*playable.LogMessage, 2)

	pot := 0
	for i, id := range playerIDs {
		p := newParticipant(id)
		logs = append(logs, playable.SimpleLogMessage(id, "{} paid the ante of ${%d}", opts.Ante))
		p.SubtractBalance(opts.Ante)
		pot += opts.Ante

		if i == 0 {
			// small blind
			pot += p.Bet(smallBlind)
			blindLogs[0] = playable.SimpleLogMessage(id, "{} paid the small blind of ${%d}", smallBlind)
		} else if i == 1 {
			// big blind
			pot += p.Bet(opts.LowerLimit)
			blindLogs[1] = playable.SimpleLogMessage(id, "{} paid the big blind of ${%d}", opts.LowerLimit)
		}

		participants[id] = p
	}

	startIndex := 0
	if len(playerIDs) > 2 {
		startIndex = 2 // start with the third player
	}

	lc := make(chan []*playable.LogMessage, 256)
	lc <- append(logs, blindLogs...)

	return &Game{
		options:            opts,
		deck:               d,
		participants:       participants,
		participantOrder:   participantOrder,
		dealerState:        DealerStateStart,
		pendingDealerState: nil,
		decisionIndex:      0,
		decisionStart:      startIndex,
		pot:                pot,
		currentBet:         opts.LowerLimit,
		community:          make(deck.Hand, 0, 5),
		logChan:            lc,
	}, nil
}

func (g *Game) dealTwoCardsToEachParticipant() error {
	if g.dealerState != DealerStateStart {
		return fmt.Errorf("cannot deal cards from state %d", g.dealerState)
	}

	for i := 0; i < 2; i++ {
		for _, id := range g.participantOrder {
			card, err := g.deck.Draw()
			if err != nil {
				return err
			}

			g.participants[id].cards.AddCard(card)
		}
	}

	g.setPendingDealerState(DealerStatePreFlopBettingRound, time.Second)
	return nil
}

func validateOptions(opts Options) error {
	if opts.Ante < anteMin {
		return fmt.Errorf("ante must be >= ${%d}", anteMin)
	}

	if opts.Ante > opts.LowerLimit {
		return errors.New("ante must be less than the lower limit")
	}

	if opts.Ante > anteMax {
		return fmt.Errorf("ante must not exceed ${%d}", anteMax)
	}

	if opts.Ante%25 > 0 {
		return errors.New("ante must be divisible by ${25}")
	}

	if opts.LowerLimit%25 > 0 {
		return errors.New("lower limit must be divisible by ${25}")
	}

	if opts.LowerLimit > lowerLimitMax {
		return fmt.Errorf("lower limit must not exceed ${%d}", lowerLimitMax)
	}

	return nil
}

// GetCurrentTurn returns the participant who is currently making a decision
// Returns an error unless the game is in a betting round
func (g *Game) GetCurrentTurn() (*Participant, error) {
	if !g.InBettingRound() {
		return nil, errors.New("not in a betting round")
	}

	n := len(g.participantOrder)
	if g.decisionIndex >= n {
		return nil, errBettingRoundIsOver
	}

	index := (g.decisionStart + g.decisionIndex) % n
	return g.participants[g.participantOrder[index]], nil
}

// InBettingRound returns true if the current state is in a betting round
func (g *Game) InBettingRound() bool {
	return g.dealerState == DealerStatePreFlopBettingRound ||
		g.dealerState == DealerStateFlopBettingRound ||
		g.dealerState == DealerStateTurnBettingRound ||
		g.dealerState == DealerStateFinalBettingRound
}

// GetBetAmount returns what the current bet can be
func (g *Game) GetBetAmount() (int, error) {
	switch g.dealerState {
	case DealerStatePreFlopBettingRound:
		return g.options.LowerLimit, nil
	case DealerStateFlopBettingRound:
		return g.options.LowerLimit, nil
	case DealerStateTurnBettingRound:
		return g.options.UpperLimit, nil
	case DealerStateFinalBettingRound:
		return g.options.UpperLimit, nil
	}

	return 0, errors.New("not in a betting round")
}

// CanBet determines if cap has been reached yet
func (g *Game) CanBet() bool {
	betAmount, err := g.GetBetAmount()
	if err != nil {
		return false
	}

	// one bet + three raises is the cap
	return g.currentBet < betAmount*4
}

func (g *Game) nextDecision() {
	g.decisionIndex++
	g.advanceToActiveParticipant()
	if g.decisionIndex == len(g.participantOrder) {
		g.setPendingDealerState(DealerState(int(g.dealerState)+1), time.Second*1)
		return
	}
}

func (g *Game) newRoundSetup() {
	g.currentBet = 0
	g.decisionStart = 0
	g.decisionIndex = 0
	g.lastAction = nil
	for _, p := range g.participants {
		p.NewRound()
	}
}

func (g *Game) advanceToActiveParticipant() {
	n := len(g.participantOrder)

	// do not advance if we are already at the end
	if g.decisionIndex == n {
		return
	}

	for {
		index := (g.decisionStart + g.decisionIndex) % n
		if !g.participants[g.participantOrder[index]].folded {
			return
		}

		g.decisionIndex++
		if g.decisionIndex >= n {
			break
		}
	}

	// if we get here, it's because all players folded at the end of the round
}
