package potmanager

import (
	"sort"
)

type tier struct {
	strength     int
	participants []Participant
}

// WinManager keeps track of who won based on hand strengths
type WinManager map[int]*tier

// NewWinManager returns a new WinManager instance
func NewWinManager() WinManager {
	return make(WinManager)
}

// AddParticipant adds a participant with the provided hand strength
func (w WinManager) AddParticipant(p Participant, handStrength int) {
	t, ok := w[handStrength]
	if !ok {
		t = &tier{
			strength:     handStrength,
			participants: make([]Participant, 0),
		}
	}

	t.participants = append(t.participants, p)
	w[handStrength] = t
}

// GetSortedTiers will return a list of tiers sorted by their hand strength
// The first element in the list is the tier with the best hand
func (w WinManager) GetSortedTiers() [][]Participant {
	tiers := make([]*tier, 0, len(w))
	for _, tier := range w {
		tiers = append(tiers, tier)
	}

	sort.Sort(sort.Reverse(sortByStrength(tiers)))

	tieredParticipants := make([][]Participant, len(tiers))
	for i, t := range tiers {
		tieredParticipants[i] = t.participants
	}

	return tieredParticipants
}

type sortByStrength []*tier

func (s sortByStrength) Len() int {
	return len(s)
}

func (s sortByStrength) Less(i, j int) bool {
	return s[i].strength < s[j].strength
}

func (s sortByStrength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
