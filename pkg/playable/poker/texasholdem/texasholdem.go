package texasholdem

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"
)

var errBettingRoundIsOver = errors.New("betting round is over")

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

	pot := 0
	for i, id := range playerIDs {
		p := newParticipant(id)
		p.SubtractBalance(opts.Ante)
		pot += opts.Ante

		if i == 0 {
			// small blind
			pot += p.Bet(smallBlind)
		} else if i == 1 {
			// big blind
			pot += p.Bet(opts.LowerLimit)
		}

		participants[id] = p
	}

	startIndex := 0
	if len(playerIDs) > 2 {
		startIndex = 2 // start with the third player
	}

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
		logChan:            make(chan []*playable.LogMessage, 256),
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
	if opts.Ante < 0 {
		return errors.New("ante must be >= 0")
	}

	if opts.Ante > opts.LowerLimit {
		return errors.New("ante must be less than the lower limit")
	}

	if opts.Ante%25 > 0 {
		return errors.New("ante must be divisible by ${25}")
	}

	if opts.LowerLimit%25 > 0 {
		return errors.New("lower limit must be divisible by ${25}")
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
	if g.decisionIndex == len(g.participantOrder) {
		g.setPendingDealerState(DealerState(int(g.dealerState)+1), time.Second*1)
		return
	}

	g.advanceToActiveParticipant()
}

func (g *Game) newRoundSetup() {
	g.currentBet = 0
	g.decisionStart = 0
	g.decisionIndex = 0
	for _, p := range g.participants {
		p.NewRound()
	}
}

func (g *Game) advanceToActiveParticipant() {
	n := len(g.participantOrder)
	for i := g.decisionIndex; i < n; i++ {
		index := (g.decisionStart + i) % n
		if !g.participants[g.participantOrder[index]].folded {
			return
		}
	}

	panic("all players folded")
}
