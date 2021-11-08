package potmanager

// Participant provides an interface for retrieving and adjusting a participants balance
type Participant interface {
	ID() int
	Balance() int
	AdjustBalance(amount int)
	SetAmountInPlay(amount int)
}

// ParticipantInPot is a participant in a pot
type ParticipantInPot struct {
	Participant
	// tableIndex is where the player is seated at the table
	tableIndex int
	// amountInPlay keeps track of how much the player is risking on the current betting round
	amountInPlay int
	isAllIn      bool
	isFolded     bool
}

// reset is called when the betting round is complete
func (p *ParticipantInPot) reset() {
	p.amountInPlay = 0
	p.SetAmountInPlay(0)
}

func (p *ParticipantInPot) adjustAmountInPlay(amount int) {
	p.amountInPlay += amount
	p.Participant.SetAmountInPlay(amount)
}

// canAct returns true if the participant can check, call, bet, raise, fold
func (p *ParticipantInPot) canAct() bool {
	return !p.isFolded && !p.isAllIn
}

type sortByTableIndex []*ParticipantInPot

func (s sortByTableIndex) Len() int {
	return len(s)
}

func (s sortByTableIndex) Less(i, j int) bool {
	return s[i].tableIndex < s[j].tableIndex
}

func (s sortByTableIndex) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
