package sevencard

// Stud is a standard game of seven-card stud
// Two face-down, four face-up, and a final face-down card with
// betting rounds after the third, fourth, fifth, sixth, and final card
type Stud struct {
}

// Name returns "Seven-Card Stud"
func (s *Stud) Name() string {
	return "Seven-Card Stud"
}
