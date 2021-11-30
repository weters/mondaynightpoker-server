package texasholdem

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"mondaynightpoker-server/pkg/playable/poker/potmanager"
	"time"
)

const anteMin = 0
const anteMax = 50
const smallBlindMin = 0
const smallBlindMax = 100
const bigBlindMax = 200

type lastAction struct {
	Action   action.Action `json:"action"`
	Amount   int           `json:"amount"`
	PlayerID int64         `json:"playerId"`
}

// Game is a game of Limit Texas Hold'em
type Game struct {
	options            Options
	deck               *deck.Deck
	participants       map[int64]*Participant
	participantOrder   []*Participant
	dealerState        DealerState
	pendingDealerState *pendingDealerState
	potManager         *potmanager.PotManager
	lastAction         *lastAction
	community          deck.Hand
	logChan            chan []*playable.LogMessage

	// if true, GetEndOfGameDetails() returns
	finished bool
}

// Options configures how Texas Hold'em is played
type Options struct {
	Variant    Variant
	Ante       int
	SmallBlind int
	BigBlind   int
}

// DefaultOptions returns the default options for Texas Hold'em
func DefaultOptions() Options {
	return Options{
		Variant:    Standard,
		Ante:       25,
		SmallBlind: 25,
		BigBlind:   50,
	}
}

// NewGame returns a new game of Texas Hold'em
func NewGame(logger logrus.FieldLogger, players []playable.Player, opts Options) (*Game, error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	if len(players) < 2 {
		return nil, errors.New("there must be at least two players")
	}

	d := deck.New()
	d.Shuffle()

	participants := make(map[int64]*Participant)
	participantOrder := make([]*Participant, len(players))

	logs := make([]*playable.LogMessage, 0)
	logs = append(logs, playable.SimpleLogMessage(0, "started a new game of %s", NameFromOptions(opts)))

	mgr := potmanager.New(opts.Ante)
	for i, player := range players {
		id := player.GetPlayerID()
		p := newParticipant(id, player.GetTableStake())
		if err := mgr.SeatParticipant(p); err != nil {
			return nil, err
		}

		participants[id] = p
		participantOrder[i] = p
		logs = append(logs, playable.SimpleLogMessage(id, "{} paid the ante of ${%d}", opts.Ante))
	}
	mgr.FinishSeatingParticipants()

	lc := make(chan []*playable.LogMessage, 256)
	lc <- logs

	return &Game{
		options:            opts,
		deck:               d,
		participants:       participants,
		participantOrder:   participantOrder,
		dealerState:        DealerStateStart,
		pendingDealerState: nil,
		community:          make(deck.Hand, 0, 5),
		logChan:            lc,
		potManager:         mgr,
	}, nil
}

func (g *Game) payBlinds() {
	sb, bb := g.potManager.PayBlinds(g.options.SmallBlind, g.options.BigBlind)

	logs := make([]*playable.LogMessage, 2)
	logs[0] = playable.SimpleLogMessage(sb.ID(), "{} paid the small blind of ${%d}", g.options.SmallBlind)
	logs[1] = playable.SimpleLogMessage(bb.ID(), "{} paid the big blind of ${%d}", g.options.BigBlind)

	g.logChan <- logs
}

func (g *Game) dealStartingCardsToEachParticipant() error {
	if g.dealerState != DealerStateStart {
		return fmt.Errorf("cannot deal cards from state %d", g.dealerState)
	}

	nCards := 2
	if g.options.Variant == Pineapple || g.options.Variant == LazyPineapple {
		nCards = 3
	}

	for i := 0; i < nCards; i++ {
		for _, p := range g.participantOrder {
			card, err := g.deck.Draw()
			if err != nil {
				return err
			}

			p.cards.AddCard(card)
		}
	}

	if g.options.Variant == Pineapple {
		g.dealerState = DealerStateDiscardRound
		g.potManager.StartDecisionRound()
		return nil
	}

	g.setPendingDealerState(DealerStatePreFlopBettingRound, time.Second)
	return nil
}

func validateAmount(desc string, number, min, max int) error {
	if number%25 > 0 {
		return fmt.Errorf("%s must be in increments of ${25}", desc)
	}

	if number < min {
		return fmt.Errorf("%s must be at least ${%d}", desc, min)
	}

	if number > max {
		return fmt.Errorf("%s must be at most ${%d}", desc, max)
	}

	return nil
}

func validateOptions(opts Options) error {
	if _, ok := validVariants[opts.Variant]; !ok {
		return fmt.Errorf("invalid variant %s", opts.Variant)
	}

	if err := validateAmount("ante", opts.Ante, anteMin, anteMax); err != nil {
		return err
	}

	if err := validateAmount("small blind", opts.SmallBlind, smallBlindMin, smallBlindMax); err != nil {
		return err
	}

	if err := validateAmount("big blind", opts.BigBlind, opts.SmallBlind, bigBlindMax); err != nil {
		return err
	}

	return nil
}

// GetCurrentTurn returns the participant who is currently making a decision
// Returns an error unless the game is in a betting round
func (g *Game) GetCurrentTurn() (*Participant, error) {
	if !g.InDecisionRound() {
		return nil, errors.New("not in a betting round")
	}

	p, err := g.potManager.GetInTurnParticipant()
	if err != nil {
		return nil, err
	}

	return g.participants[p.ID()], nil
}

// InBettingRound returns true if the current state is in a betting round
func (g *Game) InBettingRound() bool {
	return g.dealerState == DealerStatePreFlopBettingRound ||
		g.dealerState == DealerStateFlopBettingRound ||
		g.dealerState == DealerStateTurnBettingRound ||
		g.dealerState == DealerStateFinalBettingRound
}

// InDecisionRound is true if the players can make a decision such as a discard or bet
func (g *Game) InDecisionRound() bool {
	return g.InBettingRound() || g.dealerState == DealerStateDiscardRound
}

func (g *Game) newRoundSetup() {
	if err := g.potManager.NextRound(); err != nil {
		return
	}

	g.lastAction = nil

	for _, p := range g.participants {
		p.NewRound()
	}
}
