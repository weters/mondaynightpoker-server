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

	a.Equal([]*ParticipantInPot{pm.tableOrder[2]}, pm.pots[0].allInParticipants)
	a.Equal([]*ParticipantInPot{pm.tableOrder[0], pm.tableOrder[1], pm.tableOrder[3]}, pm.pots[1].allInParticipants)
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

	a.Equal([]*ParticipantInPot{pm.tableOrder[0]}, pm.pots[0].allInParticipants)
	a.Equal([]*ParticipantInPot{pm.tableOrder[2]}, pm.pots[1].allInParticipants)
	a.Equal([]*ParticipantInPot{pm.tableOrder[1]}, pm.pots[2].allInParticipants)
}

func TestNew_anteAllIn(t *testing.T) {
	a := assert.New(t)
	pm := setupPotManager(50, 25, 100, 100)
	a.Equal(2, len(pm.pots))
	a.NoError(pm.ParticipantChecks(pm.tableOrder[1]))
	a.NoError(pm.ParticipantChecks(pm.tableOrder[2]))

	a.Equal([]*ParticipantInPot{pm.tableOrder[0]}, pm.pots[0].allInParticipants)
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
	a.Equal([]*ParticipantInPot{pm.tableOrder[0]}, pm.pots[0].allInParticipants)
	a.Equal([]*ParticipantInPot{pm.tableOrder[2]}, pm.pots[1].allInParticipants)
	a.Equal([]*ParticipantInPot{pm.tableOrder[1]}, pm.pots[2].allInParticipants)
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
