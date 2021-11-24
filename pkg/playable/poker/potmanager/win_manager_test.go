package potmanager

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

func TestNewWinManager(t *testing.T) {
	a := assert.New(t)

	wm := NewWinManager()
	wm.AddParticipant(newTestParticipant(1, 100), 10)
	wm.AddParticipant(newTestParticipant(2, 100), 20)
	wm.AddParticipant(newTestParticipant(3, 100), 30)
	wm.AddParticipant(newTestParticipant(4, 100), 20)
	wm.AddParticipant(newTestParticipant(5, 100), 30)

	tiers := wm.GetSortedTiers()
	a.Equal("3-5|2-4|1", tiersToString(tiers))
}

func tiersToString(tiers [][]Participant) string {
	s := make([]string, len(tiers))
	for i, participants := range tiers {
		ids := make([]string, len(participants))
		for j, p := range participants {
			ids[j] = strconv.FormatInt(p.ID(), 10)
		}

		s[i] = strings.Join(ids, "-")
	}

	return strings.Join(s, "|")
}
