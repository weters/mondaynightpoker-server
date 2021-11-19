package littlel

import (
	"mondaynightpoker-server/pkg/playable/poker/potmanager"
	"sort"
)

// strengthTier is a tier of hand strength
type strengthTier struct {
	strength     int
	participants []potmanager.Participant
}

type tieredHands map[int]*strengthTier

func (t tieredHands) getSortedTiers() [][]potmanager.Participant {
	tiers := make([]*strengthTier, 0, len(t))
	for _, tier := range t {
		tiers = append(tiers, tier)
	}

	sort.Sort(sort.Reverse(sortByStrength(tiers)))

	tieredParticipants := make([][]potmanager.Participant, len(tiers))
	for i, tier := range tiers {
		tieredParticipants[i] = tier.participants
	}

	return tieredParticipants
}

type sortByStrength []*strengthTier

func (s sortByStrength) Len() int {
	return len(s)
}

func (s sortByStrength) Less(i, j int) bool {
	return s[i].strength < s[j].strength
}

func (s sortByStrength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
