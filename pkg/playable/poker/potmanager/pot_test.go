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

	pm := New(0)
	pm.SeatParticipant(p1, 25)
	pm.SeatParticipant(p2, 25)
	pm.SeatParticipant(p3, 25)
	pm.SeatParticipant(p4, 25)
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
