package potmanager

import (
	"errors"
	"fmt"
	"sort"
)

// ErrParticipantNotFound is an error when a participant with a provided ID cannot be found
var ErrParticipantNotFound = errors.New("participant not found")

// ErrParticipantCannotAct is an error when the participant cannot act
var ErrParticipantCannotAct = errors.New("participant cannot act")

type pot struct {
	amount       int
	participants map[*ParticipantInPot]bool
}

// PotManager provides capabilities for keeping track of bets and pots
type PotManager struct {
	participants map[int]*ParticipantInPot
	tableOrder   []*ParticipantInPot
	pots         []*pot
	// actionStartIndex is where the action started, or changed (i.e., a raise)
	actionStartIndex int
	// actionAtIndex is who is currently making a decision
	actionAtIndex int
	actionAmount  int
}

// New instantiates a new PotManager
func New(initialPot int) *PotManager {
	return &PotManager{
		participants: make(map[int]*ParticipantInPot),
		tableOrder:   make([]*ParticipantInPot, 0),
		pots: []*pot{
			{
				amount:       0,
				participants: make(map[*ParticipantInPot]bool),
			},
		},
	}
}

// SeatParticipant adds a participant to the table in the order called
// This method must be called in order of the players
func (p *PotManager) SeatParticipant(pt Participant, ante int) {
	pip := &ParticipantInPot{
		Participant: pt,
		tableIndex:  len(p.tableOrder),
	}
	p.participants[pt.ID()] = pip
	p.tableOrder = append(p.tableOrder, pip)
	p.pots[0].participants[pip] = true

	p.adjustParticipant(pip, ante)
}

// FinishSeatingParticipants must be called after all participants have been seated
func (p *PotManager) FinishSeatingParticipants() {
	p.calculatePot()
}

// ParticipantFolds handles a fold
func (p *PotManager) ParticipantFolds(pt Participant) error {
	pip, err := p.getActiveParticipantInPot(pt)
	if err != nil {
		return err
	}

	pip.isFolded = true
	p.completeTurn()
	return nil
}

// ParticipantChecks handles a check
func (p *PotManager) ParticipantChecks(pt Participant) error {
	pip, err := p.getActiveParticipantInPot(pt)
	if err != nil {
		return err
	}

	if pip.amountInPlay != p.actionAmount {
		return errors.New("participant cannot check")
	}

	p.completeTurn()
	return nil
}

// ParticipantCalls handles a call
func (p *PotManager) ParticipantCalls(pt Participant) error {
	pip, err := p.getActiveParticipantInPot(pt)
	if err != nil {
		return err
	}

	if p.actionAmount <= pip.amountInPlay {
		return fmt.Errorf("participant cannot call")
	}

	p.adjustParticipant(pip, p.actionAmount)
	p.completeTurn()
	return nil
}

// ParticipantBetsOrRaises will place a bet or a raise for a participant
// This method only enforces that the bet or raise is above the previous bet or raise. Any additional logic
// must be handled by the game.
func (p *PotManager) ParticipantBetsOrRaises(pt Participant, newBetOrRaise int) error {
	pip, err := p.getActiveParticipantInPot(pt)
	if err != nil {
		return err
	}

	if newBetOrRaise <= p.actionAmount {
		return fmt.Errorf("raise must be greater than previous bet")
	}

	if newBetOrRaise <= pip.amountInPlay {
		return fmt.Errorf("participant has more in play than the new bet or raise")
	}

	p.actionStartIndex = pip.tableIndex
	p.actionAtIndex = 0

	p.actionAmount = newBetOrRaise
	p.adjustParticipant(pip, newBetOrRaise)

	p.completeTurn()
	return nil
}

func (p *PotManager) adjustParticipant(pip *ParticipantInPot, adjustment int) {
	adjustment -= pip.amountInPlay
	if adjustment >= pip.Balance() {
		adjustment = pip.Balance()
		pip.isAllIn = true
	}

	pip.adjustAmountInPlay(adjustment)
	pip.Participant.AdjustBalance(-1 * adjustment)
}

// completeTurn must be called after a participant bets, raises, checks, calls, or folds
func (p *PotManager) completeTurn() {
	// stay in for loop until we find a player who can act
	for p.actionAtIndex++; p.actionAtIndex < len(p.tableOrder); p.actionAtIndex++ {
		pip := p.tableOrder[p.normalizedActionAtIndex()]
		// player can act
		if !pip.isAllIn && !pip.isFolded {
			return
		}
	}

	// if we reached this point, all players have acted
	p.calculatePot()
}

func (p *PotManager) calculatePot() {
	pips := make([]*ParticipantInPot, len(p.tableOrder))
	copy(pips, p.tableOrder)
	sort.Sort(SortByAmountInPlay(pips))

	currentPot := p.pots[len(p.pots)-1]

	prevAmount := 0
	newPotAtAmount := 0

	for i, pip := range pips {
		if pip.amountInPlay > prevAmount {
			if !pip.isFolded && pip.amountInPlay < p.actionAmount && newPotAtAmount < pip.amountInPlay {
				// create a new pot
				newPot := &pot{
					participants: ParticipantInPartList(pips[i:]).Map(),
				}
				currentPot = newPot
				p.pots = append(p.pots, newPot)
				newPotAtAmount = pip.amountInPlay
			}

			currentPot.amount += (len(pips) - i) * (pip.amountInPlay - prevAmount)
		}

		prevAmount = pip.amountInPlay
	}

	p.reset()
}

func (p *PotManager) reset() {
	for _, pip := range p.tableOrder {
		pip.reset()
	}

	p.actionAmount = 0
	p.actionStartIndex = 0
	p.actionAtIndex = 0
}

func (p *PotManager) normalizedActionAtIndex() int {
	return (p.actionStartIndex + p.actionAtIndex) % len(p.tableOrder)
}

// getActiveParticipantInPot returns the ParticipantInPot if the participant is on the clock, otherwise
// an error if the participant cannot act
func (p *PotManager) getActiveParticipantInPot(pt Participant) (*ParticipantInPot, error) {
	pip, ok := p.participants[pt.ID()]
	if !ok {
		return nil, ErrParticipantNotFound
	}

	if pip.tableIndex == p.normalizedActionAtIndex() {
		return pip, nil
	}

	return nil, ErrParticipantCannotAct
}
