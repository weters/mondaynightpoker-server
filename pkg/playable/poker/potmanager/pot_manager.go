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
	amount            int
	allInParticipants []*ParticipantInPot
}

// PotManager provides capabilities for keeping track of bets and pots
type PotManager struct {
	participants map[int]*ParticipantInPot
	tableOrder   []*ParticipantInPot
	ante         int
	pots         []*pot
	// actionStartIndex is where the action started, or changed (i.e., a raise)
	actionStartIndex int
	// actionAtIndex is who is currently making a decision
	actionAtIndex int
	actionAmount  int
}

// New instantiates a new PotManager
func New(ante, initialPot int) *PotManager {
	return &PotManager{
		participants: make(map[int]*ParticipantInPot),
		tableOrder:   make([]*ParticipantInPot, 0),
		ante:         ante,
		pots:         []*pot{{}},
	}
}

// SeatParticipant adds a participant to the table in the order called
// This method must be called in order of the players
func (p *PotManager) SeatParticipant(pt Participant) {
	pip := &ParticipantInPot{
		Participant: pt,
		tableIndex:  len(p.tableOrder),
	}
	p.participants[pt.ID()] = pip
	p.tableOrder = append(p.tableOrder, pip)

	p.adjustParticipant(pip, p.ante)
}

// FinishSeatingParticipants must be called after all participants have been seated
func (p *PotManager) FinishSeatingParticipants() {
	p.actionAmount = p.ante
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
	if p.actionAmount == 0 {
		p.reset()
		return
	}

	allInAmounts := make(map[int][]*ParticipantInPot)
	totalAction := 0
	for _, pip := range p.tableOrder {
		totalAction += pip.amountInPlay

		// participant went all-in this round
		if !pip.isFolded && pip.isAllIn && pip.amountInPlay > 0 {
			pips, ok := allInAmounts[pip.amountInPlay]
			if !ok {
				pips = make([]*ParticipantInPot, 0)
			}

			allInAmounts[pip.amountInPlay] = append(pips, pip)
		}
	}

	currentPot := p.pots[len(p.pots)-1]
	// if it's not nil, then there is someone all-in on this pot. create a side pot
	if currentPot.allInParticipants != nil {
		currentPot = &pot{}
		p.pots = append(p.pots, currentPot)
	}

	// no all-in
	if len(allInAmounts) == 0 {
		currentPot.amount += totalAction
		p.reset()
		return
	}

	// add the bet as the final entry to allInAmounts, even if it isn't actually an all-in
	// just don't do it if we already have a value there
	if _, ok := allInAmounts[p.actionAmount]; !ok {
		allInAmounts[p.actionAmount] = nil
	}

	amounts := make([]int, 0, len(allInAmounts))
	for amount := range allInAmounts {
		amounts = append(amounts, amount)
	}
	sort.Ints(amounts)

	prevAmount := 0
	for i, allInAmount := range amounts {
		potAmount := 0
		for _, pip := range p.tableOrder {
			amount := pip.amountInPlay
			if amount > allInAmount {
				amount = allInAmount
			}

			diffAmount := amount - prevAmount
			if diffAmount < 0 {
				diffAmount = 0
			}

			potAmount += diffAmount
		}

		currentPot.amount += potAmount
		currentPot.allInParticipants = allInAmounts[allInAmount]

		if i+1 != len(amounts) {
			currentPot = &pot{}
			p.pots = append(p.pots, currentPot)
		}

		prevAmount = allInAmount
	}

	p.reset()
}

func (p *PotManager) reset() {
	for _, pip := range p.tableOrder {
		pip.reset()
	}

	p.actionAmount = 0
	p.actionAtIndex = 0

	// reset actionStartIndex to first non-folded, non-all-in player
	for p.actionStartIndex = 0; p.actionStartIndex < len(p.tableOrder) && !p.tableOrder[p.actionStartIndex].canAct(); p.actionStartIndex++ {
		// no-op
	}
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
