package potmanager

import (
	"errors"
	"fmt"
	"sort"
)

// ParticipantError is an error that happened because of a participant error
type ParticipantError string

func (p ParticipantError) Error() string {
	return string(p)
}

func newParticipantError(format string, a ...interface{}) ParticipantError {
	return ParticipantError(fmt.Sprintf(format, a...))
}

// ErrGameOver is an error an action is attempted after the game ended
var ErrGameOver = errors.New("game is over")

// ErrRoundOver is an error when the round is over
var ErrRoundOver = errors.New("round is over")

// ErrParticipantNotFound is an error when a participant with a provided ID cannot be found
var ErrParticipantNotFound = errors.New("participant not found")

// ErrParticipantCannotAct is an error when the participant cannot act
var ErrParticipantCannotAct = ParticipantError("it is not your turn")

type participantInPotMap map[*participantInPot]bool

type pot struct {
	amount            int
	allInParticipants participantInPotMap
}

// PotManager provides capabilities for keeping track of bets and pots
type PotManager struct {
	participants map[int64]*participantInPot
	tableOrder   []*participantInPot
	ante         int
	pots         []*pot
	// actionStartIndex is where the action started, or changed (i.e., a raise)
	actionStartIndex int
	// actionAtIndex is who is currently making a decision
	actionAtIndex int
	actionAmount  int
	// amountInPlay is how much has been bet or called, but not yet added to the pot
	amountInPlay int

	// needsPotCalculation should be set to true if we need to recalculate the pot
	needsPotCalculation bool

	// isGameOver will prevent any further action from happening
	isGameOver bool
}

// New instantiates a new PotManager
func New(ante int) *PotManager {
	return &PotManager{
		participants: make(map[int64]*participantInPot),
		tableOrder:   make([]*participantInPot, 0),
		ante:         ante,
		pots:         []*pot{{}},
	}
}

// SeatParticipant adds a participant to the table in the order called
// This method must be called in order of the players
func (p *PotManager) SeatParticipant(pt Participant) error {
	if pt.Balance() <= 0 {
		return errors.New("cannot seat participant without a balance")
	}

	pip := &participantInPot{
		Participant: pt,
		tableIndex:  len(p.tableOrder),
	}
	p.participants[pt.ID()] = pip
	p.tableOrder = append(p.tableOrder, pip)

	p.adjustParticipant(pip, p.ante)

	return nil
}

// FinishSeatingParticipants must be called after all participants have been seated
func (p *PotManager) FinishSeatingParticipants() {
	p.actionAmount = p.ante

	p.needsPotCalculation = true
	p.calculatePot()
	p.reset()
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
		return newParticipantError("you cannot check with an active bet")
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
		return newParticipantError("you cannot call without an active bet")
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
		return newParticipantError("your raise of ${%d} must be greater than the previous bet of ${%d}", newBetOrRaise, p.actionAmount)
	}

	if newBetOrRaise <= pip.amountInPlay {
		return fmt.Errorf("participant has more in play than the new bet or raise")
	}

	if newBetOrRaise > pip.amountInPlay+pip.Balance() {
		return errors.New("bet exceeds participant's total")
	}

	p.actionStartIndex = pip.tableIndex
	p.actionAtIndex = 0

	p.actionAmount = newBetOrRaise
	p.adjustParticipant(pip, newBetOrRaise)

	p.completeTurn()
	return nil
}

// AdvanceDecision will advance a decision without taking an explicit action
func (p *PotManager) AdvanceDecision() error {
	if p.GetInTurnParticipant() == nil {
		return ErrRoundOver
	}

	p.completeTurn()
	return nil
}

// IsParticipantYetToAct returns true if the participant is not in turn and the participant has yet to act
// This also ensures the participant didn't fold and they are not all-in
func (p *PotManager) IsParticipantYetToAct(pt Participant) bool {
	pip, ok := p.participants[pt.ID()]
	if !ok {
		return false
	}

	// did they fold or go all-in
	if !pip.canAct() {
		return false
	}

	// simple formula to see if the player isn't in turn, but they are still yet to act
	check := pip.tableIndex
	if check < p.actionStartIndex {
		check += len(p.tableOrder)
	}

	return check > p.actionStartIndex+p.actionAtIndex
}

// GetCanActParticipantCount returns the number of participants in the hand who didn't fold or go all-in
func (p *PotManager) GetCanActParticipantCount() int {
	count := 0
	for _, pt := range p.tableOrder {
		if pt.canAct() {
			count++
		}
	}

	return count
}

func (p *PotManager) adjustParticipant(pip *participantInPot, adjustment int) {
	adjustment -= pip.amountInPlay
	if adjustment >= pip.Balance() {
		adjustment = pip.Balance()
		pip.isAllIn = true
	}

	p.amountInPlay += adjustment
	pip.adjustAmountInPlay(adjustment)
	pip.Participant.AdjustBalance(-1 * adjustment)
}

// GetBet returns the current bet
func (p *PotManager) GetBet() int {
	return p.actionAmount
}

// IsRoundOver returns true if all eligible participants have acted
func (p *PotManager) IsRoundOver() bool {
	return p.actionAtIndex >= len(p.tableOrder)
}

// GetInTurnParticipant returns the participant who is to act next
// Returns nil if the round is over
func (p *PotManager) GetInTurnParticipant() Participant {
	if p.IsRoundOver() {
		return nil
	}

	return p.tableOrder[p.normalizedActionAtIndex()].Participant
}

// GetPotLimitMaxBet returns the maximum bet allowed in a pot-limit game
// Pot + all previous bets + amount to call
func (p *PotManager) GetPotLimitMaxBet() int {
	previousBet := p.actionAmount

	pip := p.tableOrder[p.normalizedActionAtIndex()]
	amountToCall := p.actionAmount - pip.amountInPlay

	potTotal := p.Pots().Total() + p.amountInPlay + amountToCall

	return previousBet + potTotal
}

// Pots returns a list of pots
func (p *PotManager) Pots() Pots {
	pots := make([]*Pot, len(p.pots))
	for i, pot := range p.pots {
		a := make([]Participant, 0, len(pot.allInParticipants))
		for pip := range pot.allInParticipants {
			a = append(a, pip.Participant)
		}

		pots[i] = &Pot{
			Amount:            pot.amount,
			AllInParticipants: a,
		}
	}

	return pots
}

// PayWinners will adjust balance for the winners and return the final payouts
func (p *PotManager) PayWinners(winners [][]Participant) (map[Participant]int, error) {
	if !p.isGameOver {
		return nil, errors.New("game is not over")
	}

	p.calculatePot()

	pots := make([]*pot, len(p.pots))

	// shallow-copy
	for i, pot := range p.pots {
		tmp := *pot
		pots[i] = &tmp
	}

	payouts := make(map[Participant]int)

MainLoop:
	for _, winnerGroup := range winners {
		// convert to list of participantInPot objects. Sort by the table order
		// to ensure we pay left of dealer any uneven amounts
		pipWinnerGroup := make([]*participantInPot, len(winnerGroup))
		for i, winner := range winnerGroup {
			pipWinnerGroup[i] = p.participants[winner.ID()]
		}
		sort.Sort(sortByTableIndex(pipWinnerGroup))

		for potIndex, pot := range pots {
			if pot.amount == 0 {
				continue
			}

			// remove any users who went all in
			tmp := make([]*participantInPot, 0, len(pipWinnerGroup))
			for i, winner := range pipWinnerGroup {
				roundedWinnings := (pot.amount / 25 / len(pipWinnerGroup)) * 25
				if i < (pot.amount/25)%len(pipWinnerGroup) {
					roundedWinnings += 25
				}

				winner.AdjustBalance(roundedWinnings)
				payout := payouts[winner.Participant]
				payouts[winner.Participant] = payout + roundedWinnings

				if _, ok := pot.allInParticipants[winner]; ok {
					continue
				}

				tmp = append(tmp, winner)
			}
			pipWinnerGroup = tmp
			pot.amount = 0

			if potIndex+1 == len(pots) {
				break MainLoop
			} else if len(pipWinnerGroup) == 0 {
				break
			}
		}
	}

	return payouts, nil
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
	p.needsPotCalculation = true
}

func (p *PotManager) calculatePot() {
	if !p.needsPotCalculation {
		return
	}

	p.needsPotCalculation = false

	if p.actionAmount == 0 {
		return
	}

	allInAmounts := make(map[int]map[*participantInPot]bool)
	totalAction := 0
	for _, pip := range p.tableOrder {
		totalAction += pip.amountInPlay

		// participant went all-in this round
		if !pip.isFolded && pip.isAllIn && pip.amountInPlay > 0 {
			pips, ok := allInAmounts[pip.amountInPlay]
			if !ok {
				pips = make(map[*participantInPot]bool)
				allInAmounts[pip.amountInPlay] = pips
			}

			pips[pip] = true
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
}

// NextRound advances to the next round
func (p *PotManager) NextRound() error {
	if !p.IsRoundOver() {
		return errors.New("round is not over")
	}

	p.calculatePot()
	p.reset()
	return nil
}

func (p *PotManager) reset() {
	for _, pip := range p.tableOrder {
		pip.reset()
	}

	p.actionAmount = 0
	p.amountInPlay = 0
	p.actionAtIndex = 0

	// reset actionStartIndex to first non-folded, non-all-in player
	for p.actionStartIndex = 0; p.actionStartIndex < len(p.tableOrder) && !p.tableOrder[p.actionStartIndex].canAct(); p.actionStartIndex++ {
		// no-op
	}
}

func (p *PotManager) normalizedActionAtIndex() int {
	return (p.actionStartIndex + p.actionAtIndex) % len(p.tableOrder)
}

// getActiveParticipantInPot returns the participantInPot if the participant is on the clock, otherwise
// an error if the participant cannot act
func (p *PotManager) getActiveParticipantInPot(pt Participant) (*participantInPot, error) {
	if p.isGameOver {
		return nil, ErrGameOver
	}

	pit := p.GetInTurnParticipant()
	if pit == nil {
		return nil, ErrRoundOver
	}

	if pit.ID() != pt.ID() {
		return nil, ErrParticipantCannotAct
	}

	pip, ok := p.participants[pt.ID()]
	if !ok {
		panic("participant not found")
	}

	return pip, nil
}

// EndGame will prevent further action from happening
func (p *PotManager) EndGame() {
	p.isGameOver = true
}
