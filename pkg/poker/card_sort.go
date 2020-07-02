package poker

import "mondaynightpoker-server/pkg/deck"

type sortByRank []*deck.Card

func (s sortByRank) Len() int {
	return len(s)
}

func (s sortByRank) Less(i, j int) bool {
	return s[i].Rank < s[j].Rank
}

func (s sortByRank) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
