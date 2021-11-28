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

// ErrInDecisionRound is an error when a participant tries to check, call, bet, raise, or fold while in the decision round
var ErrInDecisionRound = errors.New("you cannot perform a betting action while in a decision round")

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
	// actionDiffAmount keeps track of the raise
	actionDiffAmount int
	// amountInPlay is how much has been bet or called, but not yet added to the pot
	amountInPlay int

	// needsPotCalculation should be set to true if we need to recalculate the pot
	needsPotCalculation bool

	// isGameOver will prevent any further action from happening
	isGameOver bool

	// isInDecisionRound allows all non-folded participants to do something
	isInDecisionRound bool
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
	if p.isInDecisionRound {
		return ErrInDecisionRound
	}

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
	if p.isInDecisionRound {
		return ErrInDecisionRound
	}

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
	if p.isInDecisionRound {
		return ErrInDecisionRound
	}

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

// PayBlinds will have the participants pay the blinds
func (p *PotManager) PayBlinds(sbAmt, bbAmt int) (smallBlind Participant, bigBlind Participant) {
	// Reminder: the dealer is the last player in tableOrder

	if bbAmt < sbAmt {
		panic(fmt.Sprintf("big blind (%d) must be more than small blind (%d)", bbAmt, sbAmt))
	}

	var sbPip, bbPip *participantInPot

	if len(p.tableOrder) == 2 {
		// dealer is small blind
		sbPip = p.tableOrder[1]
		bbPip = p.tableOrder[0]
		p.actionStartIndex = 1
	} else {
		sbPip = p.tableOrder[0]
		bbPip = p.tableOrder[1]
		p.actionStartIndex = 2
	}

	p.actionAmount = bbAmt
	p.actionDiffAmount = bbAmt
	p.adjustParticipant(sbPip, sbAmt)
	p.adjustParticipant(bbPip, bbAmt)

	return sbPip, bbPip
}

// ParticipantBetsOrRaises will place a bet or a raise for a participant
// This method only enforces that the bet or raise is above the previous bet or raise. Any additional logic
// must be handled by the game.
func (p *PotManager) ParticipantBetsOrRaises(pt Participant, newBetOrRaise int) error {
	if p.isInDecisionRound {
		return ErrInDecisionRound
	}

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

	p.actionDiffAmount = newBetOrRaise - p.actionAmount
	p.actionAmount = newBetOrRaise
	p.adjustParticipant(pip, newBetOrRaise)

	p.completeTurn()
	return nil
}

// GetParticipantAllInAmount returns the amount required for the participant to go all-in
func (p *PotManager) GetParticipantAllInAmount(pt Participant) int {
	pip, ok := p.participants[pt.ID()]
	if !ok {
		panic("participant not found")
	}

	return pip.amountInPlay + pip.Balance()
}

// AdvanceDecision will advance a decision without taking an explicit action
func (p *PotManager) AdvanceDecision() error {
	if _, err := p.GetInTurnParticipant(); err != nil {
		return err
	}

	p.completeTurn()
	return nil
}

// StartDecisionRound starts a decision round which is a round that all non-folded players (including all-in) can participate
func (p *PotManager) StartDecisionRound() {
	p.isInDecisionRound = true
	p.reset()
}

// IsParticipantYetToAct returns true if the participant is not in turn and the participant has yet to act
// This also ensures the participant didn't fold, and they are not all-in
func (p *PotManager) IsParticipantYetToAct(pt Participant) bool {
	if p.isGameOver {
		return false
	}

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

// GetAliveParticipantCount returns the number of participants who haven't folded
func (p *PotManager) GetAliveParticipantCount() int {
	count := 0
	for _, pt := range p.tableOrder {
		if !pt.isFolded {
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

// GetRaise returns the raise amount
// Example. Player A bets $25. Player B raises to $50. This would return $25.
func (p *PotManager) GetRaise() int {
	return p.actionDiffAmount
}

// IsRoundOver returns true if all eligible participants have acted
func (p *PotManager) IsRoundOver() bool {
	return p.actionAtIndex >= len(p.tableOrder)
}

// GetInTurnParticipant returns the participant who is to act next
// Returns nil if the round is over
func (p *PotManager) GetInTurnParticipant() (Participant, error) {
	if p.isGameOver {
		return nil, ErrGameOver
	}

	if p.IsRoundOver() {
		return nil, ErrRoundOver
	}

	return p.tableOrder[p.normalizedActionAtIndex()].Participant, nil
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

// GetTotalOnTable returns the total of all the money in all the pots and any money currently in the bet
func (p *PotManager) GetTotalOnTable() int {
	return p.amountInPlay + p.Pots().Total()
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

		if p.isInDecisionRound {
			if !pip.isFolded {
				return
			}
		} else {
			if pip.canAct() {
				return
			}
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
	liveParticipants := make(map[*participantInPot]bool)

	for _, pip := range p.tableOrder {
		if pip.canAct() {
			liveParticipants[pip] = true
		}

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

	switch len(liveParticipants) {
	case 1:
		var pip *participantInPot
		for aPip := range liveParticipants {
			pip = aPip
			break
		}

		if currentPot.allInParticipants == nil {
			pip.adjustAmountInPlay(-1 * currentPot.amount)
			pip.Participant.AdjustBalance(currentPot.amount)
			p.pots = p.pots[0 : len(p.pots)-1]
		}
	case 0:
		if len(currentPot.allInParticipants) == 1 {
			var pip *participantInPot
			for aPip := range currentPot.allInParticipants {
				pip = aPip
				break
			}

			pip.adjustAmountInPlay(-1 * currentPot.amount)
			pip.Participant.AdjustBalance(currentPot.amount)
			p.pots = p.pots[0 : len(p.pots)-1]
		}
	}
}

// NextRound advances to the next round
func (p *PotManager) NextRound() error {
	if !p.IsRoundOver() {
		return errors.New("round is not over")
	}

	p.calculatePot()

	p.isInDecisionRound = false
	p.reset()
	return nil
}

func (p *PotManager) reset() {
	for _, pip := range p.tableOrder {
		pip.reset()
	}

	p.actionAmount = 0
	p.actionDiffAmount = 0
	p.amountInPlay = 0
	p.actionAtIndex = 0

	// set the appropriate actionStartIndex
	// if there's only one valid participant left (i.e., everyone else folded or is all-in) then skip to the end

	firstValid := -1
	totalValid := 0
	for i, pip := range p.tableOrder {
		var isValid bool
		if p.isInDecisionRound {
			isValid = !pip.isFolded
		} else {
			isValid = pip.canAct()
		}

		if isValid {
			if firstValid == -1 {
				firstValid = i
			}

			totalValid++
		}
	}

	if totalValid > 1 {
		p.actionStartIndex = firstValid
	} else {
		// move the pointer to the end to force round over
		p.actionStartIndex = 0
		p.actionAtIndex = len(p.tableOrder)
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

	pit, err := p.GetInTurnParticipant()
	if err != nil {
		return nil, err
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
