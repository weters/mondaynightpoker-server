package potmanager

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type testParticipant struct {
	id           int
	balance      int
	amountInPlay int
}

func (t *testParticipant) ID() int {
	return t.id
}

func (t *testParticipant) Balance() int {
	return t.balance
}

func (t *testParticipant) AdjustBalance(amount int) {
	t.balance += amount
}

func (t *testParticipant) SetAmountInPlay(amount int) {
	t.amountInPlay = amount
}

func newTestParticipant(id, balance int) *testParticipant {
	return &testParticipant{
		id:      id,
		balance: balance,
	}
}

func TestNew_smokeTest(t *testing.T) {
	a := assert.New(t)

	p1 := newTestParticipant(1, 100)
	p2 := newTestParticipant(2, 100)
	p3 := newTestParticipant(3, 100)
	p4 := newTestParticipant(4, 125)

	pm := New(25, 0)
	pm.SeatParticipant(p1)
	pm.SeatParticipant(p2)
	pm.SeatParticipant(p3)
	pm.SeatParticipant(p4)
	pm.FinishSeatingParticipants() // pot is at 100

	a.Equal(1, len(pm.Pots()))

	a.Equal(75, p1.balance)

	a.EqualError(pm.ParticipantCalls(p1), "participant cannot call")
	a.NoError(pm.ParticipantBetsOrRaises(p1, 25)) // 125
	a.Equal(ErrParticipantCannotAct, pm.ParticipantBetsOrRaises(p1, 50))

	a.EqualError(pm.ParticipantBetsOrRaises(p2, 25), "raise must be greater than previous bet")
	a.EqualError(pm.ParticipantChecks(p2), "participant cannot check")
	a.NoError(pm.ParticipantCalls(p2)) // 150

	a.NoError(pm.ParticipantBetsOrRaises(p3, 50)) // 200
	a.NoError(pm.ParticipantCalls(p4))            // 250
	a.NoError(pm.ParticipantFolds(p1))
	a.NoError(pm.ParticipantCalls(p2)) // 275
	a.Equal(ErrParticipantCannotAct, pm.ParticipantCalls(p3))

	a.Equal(1, len(pm.pots))
	a.Equal(275, pm.pots[0].amount)
}

func TestNew_simpleAllIn(t *testing.T) {
	a := assert.New(t)

	pm := setupPotManager(0, 10, 10, 5, 10)
	a.NoError(pm.ParticipantBetsOrRaises(pm.tableOrder[0], 10))
	a.NoError(pm.ParticipantCalls(pm.tableOrder[1]))
	a.NoError(pm.ParticipantCalls(pm.tableOrder[2]))
	a.NoError(pm.ParticipantCalls(pm.tableOrder[3]))

	a.Equal(2, len(pm.pots))
	a.Equal(20, pm.pots[0].amount)
	a.Equal(15, pm.pots[1].amount)

	a.Equal(participantInPotMap{pm.tableOrder[2]: true}, pm.pots[0].allInParticipants)
	a.Equal(participantInPotMap{
		pm.tableOrder[0]: true,
		pm.tableOrder[1]: true,
		pm.tableOrder[3]: true,
	}, pm.pots[1].allInParticipants)

	pots := pm.Pots()
	a.Equal(2, len(pots))
	a.Equal(20, pots[0].Amount)
	a.Equal(1, len(pots[0].AllInParticipants))
	a.Equal(15, pots[1].Amount)
	a.Equal(3, len(pots[1].AllInParticipants))
}

func TestNew_complexAllIn(t *testing.T) {
	a := assert.New(t)

	pm := setupPotManager(0, 5, 15, 10, 20)
	a.NoError(pm.ParticipantBetsOrRaises(pm.tableOrder[0], 5))
	a.NoError(pm.ParticipantBetsOrRaises(pm.tableOrder[1], 10))
	a.NoError(pm.ParticipantCalls(pm.tableOrder[2]))
	a.NoError(pm.ParticipantBetsOrRaises(pm.tableOrder[3], 15))
	a.NoError(pm.ParticipantCalls(pm.tableOrder[1]))

	a.Equal(3, len(pm.pots))
	a.Equal(20, pm.pots[0].amount)
	a.Equal(15, pm.pots[1].amount)
	a.Equal(10, pm.pots[2].amount)

	a.Equal(0, pm.tableOrder[0].Balance())
	a.Equal(0, pm.tableOrder[1].Balance())
	a.Equal(0, pm.tableOrder[2].Balance())
	a.Equal(5, pm.tableOrder[3].Balance())

	a.Equal(participantInPotMap{pm.tableOrder[0]: true}, pm.pots[0].allInParticipants)
	a.Equal(participantInPotMap{pm.tableOrder[2]: true}, pm.pots[1].allInParticipants)
	a.Equal(participantInPotMap{pm.tableOrder[1]: true}, pm.pots[2].allInParticipants)
}

func TestNew_anteAllIn(t *testing.T) {
	a := assert.New(t)
	pm := setupPotManager(50, 25, 100, 100)
	a.Equal(2, len(pm.pots))
	a.NoError(pm.ParticipantChecks(pm.tableOrder[1]))
	a.NoError(pm.ParticipantChecks(pm.tableOrder[2]))

	a.Equal(participantInPotMap{pm.tableOrder[0]: true}, pm.pots[0].allInParticipants)
	a.Nil(pm.pots[1].allInParticipants)
}

func TestNew_multiRoundWithAllIn(t *testing.T) {
	a := assert.New(t)
	pm := setupPotManager(5, 5, 15, 10, 20)

	// round 1
	a.NoError(pm.ParticipantBetsOrRaises(pm.tableOrder[1], 5))
	a.NoError(pm.ParticipantCalls(pm.tableOrder[2]))
	a.NoError(pm.ParticipantCalls(pm.tableOrder[3]))

	// round 2
	a.NoError(pm.ParticipantBetsOrRaises(pm.tableOrder[1], 5))
	a.NoError(pm.ParticipantCalls(pm.tableOrder[3]))

	a.Equal(3, len(pm.pots))
	a.Equal(participantInPotMap{pm.tableOrder[0]: true}, pm.pots[0].allInParticipants)
	a.Equal(participantInPotMap{pm.tableOrder[2]: true}, pm.pots[1].allInParticipants)
	a.Equal(participantInPotMap{pm.tableOrder[1]: true}, pm.pots[2].allInParticipants)
}

func TestPotManager_PayWinners_oneWinner(t *testing.T) {
	pm := setupPotManager(25, 25, 25, 25)
	payouts := pm.PayWinners([][]Participant{
		{pm.tableOrder[0].Participant},
	})

	a := assert.New(t)
	a.Equal(map[Participant]int{
		pm.tableOrder[0].Participant: 75,
	}, payouts)
}

func TestPotManager_PayWinners_twoWinner(t *testing.T) {
	pm := setupPotManager(25, 25, 25, 25)
	payouts := pm.PayWinners([][]Participant{
		{pm.tableOrder[0].Participant, pm.tableOrder[1].Participant},
	})

	a := assert.New(t)
	a.Equal(map[Participant]int{
		pm.tableOrder[0].Participant: 50,
		pm.tableOrder[1].Participant: 25,
	}, payouts)
}

func TestPotManager_PayWinners_simpleAllIn(t *testing.T) {
	pm := setupPotManager(50, 25, 50, 50)
	payouts := pm.PayWinners([][]Participant{
		{pm.tableOrder[0].Participant}, // can only win 75
		{pm.tableOrder[1].Participant}, // wins remaining
		{pm.tableOrder[2].Participant}, // shouldn't win any
	})

	a := assert.New(t)
	a.Equal(map[Participant]int{
		pm.tableOrder[0].Participant: 75,
		pm.tableOrder[1].Participant: 50,
	}, payouts)

	a.Equal(75, pm.tableOrder[0].Balance())
	a.Equal(50, pm.tableOrder[1].Balance())
	a.Equal(0, pm.tableOrder[2].Balance())
}

func TestPotManager_PayWinners_complexAllIn(t *testing.T) {
	pm := setupPotManager(75, 25, 50, 50, 75, 75) // 275
	payouts := pm.PayWinners([][]Participant{
		{
			pm.tableOrder[0].Participant,
			pm.tableOrder[1].Participant,
		},
		{
			pm.tableOrder[2].Participant,
			pm.tableOrder[3].Participant,
		},
	})

	a := assert.New(t)
	a.Equal(125, pm.pots[0].amount)
	a.Equal(100, pm.pots[1].amount)
	a.Equal(50, pm.pots[2].amount)

	a.Equal(map[Participant]int{
		pm.tableOrder[0].Participant: 75,
		pm.tableOrder[1].Participant: 150,
		pm.tableOrder[2].Participant: 25,
		pm.tableOrder[3].Participant: 25,
	}, payouts)
}

func setupPotManager(ante int, balances ...int) *PotManager {
	pm := New(ante, 0)
	for i, balance := range balances {
		p := newTestParticipant(i+1, balance)
		pm.SeatParticipant(p)
	}
	pm.FinishSeatingParticipants()
	return pm
}
