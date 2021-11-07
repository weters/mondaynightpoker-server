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

// SortByAmountInPlay sorts a list of players by amountInPlay
type SortByAmountInPlay []*ParticipantInPot

func (s SortByAmountInPlay) Len() int {
	return len(s)
}

func (s SortByAmountInPlay) Less(i, j int) bool {
	return s[i].amountInPlay < s[j].amountInPlay
}

func (s SortByAmountInPlay) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// ParticipantInPartList is a list of ParticipantInPot
type ParticipantInPartList []*ParticipantInPot

// ActiveMap returns a map of participants that didn't fold
func (p ParticipantInPartList) ActiveMap() map[*ParticipantInPot]bool {
	m := make(map[*ParticipantInPot]bool)
	for _, pip := range p {
		if !pip.isFolded {
			m[pip] = true
		}
	}

	return m
}
