package handanalyzer

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func Test_straightTracker_popThroughWilds_noWildAtEnd(t *testing.T) {
	st := &straightTracker{streak: deck.CardsFromString("5c,!4d,!3d")}
	assert.PanicsWithValue(t, "reclaimWilds() must not be called if the last card in the streak is a wild", func() {
		st.reclaimWilds(2, 1)
	})
}

func Test_straightTracker_popThroughWilds_ensureEnoughWilds(t *testing.T) {
	st := &straightTracker{streak: deck.CardsFromString("5c,!4d,!3d,2c")}
	assert.PanicsWithValue(t, "reclaimWilds() must only be called if we can reclaim enough wilds to cover the gap", func() {
		st.reclaimWilds(2, 3)
	})
}

func Test_straightTracker_popThroughWilds_checkImpossibleStreak(t *testing.T) {
	// this streak is already a straight, so we shouldn't have to reclaim cards
	st := &straightTracker{streak: deck.CardsFromString("9c,!8d,7c,!6d,5d")}
	st.usedWilds = 2
	assert.PanicsWithValue(t, "impossible 3 or 5-card streak found", func() {
		st.reclaimWilds(2, 2)
	})
}

func Test_randomStraights(t *testing.T) {
	// run through a bunch of random hands and ensure that
	// we don't hit any of our panics
	a := assert.New(t)
	nStraights := 0
	for i := 0; i < 100000; i++ {
		h := make(deck.Hand, 0)
		d := deck.New()
		d.Shuffle()

		for i := 0; i < 7; i++ {
			card, err := d.Draw()
			a.NoError(err)
			h.AddCard(card)

			nWild := rand.Intn(5)

			if i < nWild {
				card.IsWild = true
			}
		}

		ha := New(5, h)
		_, ok := ha.GetStraight()
		if ok {
			nStraights++
		}
	}

	t.Logf("found %d straights", nStraights)
}
