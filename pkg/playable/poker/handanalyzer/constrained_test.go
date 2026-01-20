package handanalyzer

import (
	"mondaynightpoker-server/pkg/deck"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateWildCombinations(t *testing.T) {
	a := assert.New(t)

	// 0 wilds → 1 empty combination
	wilds := deck.Hand{}
	combos := generateWildCombinations(wilds)
	a.Len(combos, 1)
	a.Len(combos[0], 0)

	// 1 wild → 2 combinations (RankMode, SuitMode)
	wilds = deck.Hand{deck.CardFromString("!5c")}
	combos = generateWildCombinations(wilds)
	a.Len(combos, 2)
	// Check both modes are present
	modes := make(map[WildMode]bool)
	for _, c := range combos {
		a.Len(c, 1)
		modes[c[0].Mode] = true
	}
	a.True(modes[RankMode])
	a.True(modes[SuitMode])

	// 2 wilds → 4 combinations
	wilds = deck.Hand{deck.CardFromString("!5c"), deck.CardFromString("!6d")}
	combos = generateWildCombinations(wilds)
	a.Len(combos, 4)

	// 3 wilds → 8 combinations
	wilds = deck.Hand{deck.CardFromString("!5c"), deck.CardFromString("!6d"), deck.CardFromString("!7h")}
	combos = generateWildCombinations(wilds)
	a.Len(combos, 8)
}

func TestConstrainedFlush(t *testing.T) {
	a := assert.New(t)

	// Test: Wild in suit mode can contribute to any suit's flush
	// 2c, 4c, 6c, 8c, !10d - with suit mode, the 10d keeps its rank (10) but changes suit
	// For a flush, suit-mode wilds contribute their rank to the flush
	h := New(5, deck.CardsFromString("2c,4c,6c,8c,!10d"))
	flush, ok := h.GetFlush()
	a.True(ok, "Should make a flush with suit-mode wild")
	a.Contains(flush, 10, "Suit-mode wild contributes its rank")

	// Test: Rank-mode wild can only contribute to its original suit
	// If the wild is !10d (diamond), in rank-mode it can only help a diamond flush
	h = New(5, deck.CardsFromString("2c,4c,6d,8c,!10d"))
	_, ok = h.GetFlush()
	// This should not be a flush because:
	// - We have 4 clubs (2c, 4c, 8c) - need 1 more
	// - rank-mode !10d can only help diamonds, not clubs
	// - suit-mode !10d can help clubs but only provides rank 10
	// Best case: suit-mode wild makes a club flush with 10, 8, 4, 2 + something
	// Actually we only have 3 clubs (2c, 4c, 8c), so we need the wild to be suit-mode
	a.False(ok, "Only 3 clubs + 1 diamond + wild can't make a flush without 5 of a suit")

	// Test: More clubs to test flush detection
	h = New(5, deck.CardsFromString("2c,4c,6c,8c,!10c"))
	_, ok = h.GetFlush()
	a.True(ok, "Should make a flush with same-suit wild")
}

func TestConstrainedStraight(t *testing.T) {
	a := assert.New(t)

	// Test: Rank-mode wild can fill any gap in a straight
	// 3c, 4d, 5h, 7s, !6c (wild is a 6) - rank mode wild fills the 6 gap
	h := New(5, deck.CardsFromString("3c,4d,5h,7s,!6c"))
	straight, ok := h.GetStraight()
	a.True(ok, "Rank-mode wild can fill the gap")
	a.Equal(7, straight, "Straight should be 7-high")

	// Test: Suit-mode wild can only occupy its original rank position
	// 3c, 4d, 5h, 7s, !9c - the wild is a 9, in suit mode it can only be a 9
	// This cannot make a straight because we need a 6, not a 9
	h = New(5, deck.CardsFromString("3c,4d,5h,7s,!9c"))
	straight, ok = h.GetStraight()
	// Best combination: rank mode wild fills gap at 6 → 3-4-5-6-7 straight
	a.True(ok, "Rank-mode wild can fill the 6 gap")
	a.Equal(7, straight)

	// Test: Wild at the right position for suit mode
	// 3c, 4d, 5h, 7s, !6c - wild is 6, can fill gap in either mode
	h = New(5, deck.CardsFromString("3c,4d,5h,7s,!6c"))
	straight, ok = h.GetStraight()
	a.True(ok)
	a.Equal(7, straight)
}

func TestConstrainedStraightFlush(t *testing.T) {
	a := assert.New(t)

	// Test: 2h, 3h, 4h, 5h, !6s
	// Suit mode: 6 becomes heart → Straight flush A-2-3-4-5 or 2-3-4-5-6
	// Wait, the wild is 6s, in suit mode it keeps rank 6 but can be any suit
	// So it can be 6h, making a straight flush 2h-3h-4h-5h-6h
	h := New(5, deck.CardsFromString("2h,3h,4h,5h,!6s"))
	sf, ok := h.GetStraightFlush()
	a.True(ok, "Suit-mode wild can create straight flush")
	a.Equal(6, sf, "6-high straight flush")

	// Test: 5h, 6h, 8h, 9h, !7s
	// Rank mode: 7 can be any rank, but keeps spade suit → can't help heart straight flush
	// Suit mode: 7 stays as 7, can become heart → fills the 7 gap in hearts
	h = New(5, deck.CardsFromString("5h,6h,8h,9h,!7s"))
	sf, ok = h.GetStraightFlush()
	a.True(ok, "Suit-mode wild can fill the gap")
	a.Equal(9, sf, "9-high straight flush")

	// Test: 5h, 6h, 8h, 9h, !10s - wild is wrong rank for the gap
	// The gap is at 7, but the wild is 10
	// Rank mode: wild can be 7, but stays spade → can't help heart straight flush
	// Suit mode: wild can be heart, but stays at rank 10 → doesn't fill the 7 gap
	h = New(5, deck.CardsFromString("5h,6h,8h,9h,!10s"))
	_, ok = h.GetStraightFlush()
	a.False(ok, "Wild can't help - wrong rank for suit mode, wrong suit for rank mode")
}

func TestConstrainedPairs(t *testing.T) {
	a := assert.New(t)

	// Test: 10c, 10d, 10h, 5s, !2s
	// Rank mode: wild becomes 10 → Four of a kind
	h := New(5, deck.CardsFromString("10c,10d,10h,5s,!2s"))
	fk, ok := h.GetFourOfAKind()
	a.True(ok, "Rank-mode wild makes four of a kind")
	a.Equal(10, fk)

	// Test: 10c, 10d, 5s, 5h, !2s
	// Rank mode: wild becomes 10 → Full house (10s over 5s)
	h = New(5, deck.CardsFromString("10c,10d,5s,5h,!2s"))
	fh, ok := h.GetFullHouse()
	a.True(ok, "Rank-mode wild makes full house")
	a.Equal([]int{10, 5}, fh)

	// Test: Suit-mode wild only helps its original rank
	// 10c, 10d, 5s, 5h, !5d - in suit mode, wild adds to 5s count
	h = New(5, deck.CardsFromString("10c,10d,5s,5h,!5d"))
	// In suit mode, the wild becomes a 5 of any suit → three 5s
	// But rank mode could make it a 10 → three 10s (better)
	trips, ok := h.GetThreeOfAKind()
	a.True(ok)
	a.Equal(10, trips, "Rank-mode is chosen because it gives better hand")
}

func TestConstrainedHandSelection(t *testing.T) {
	testCases := []struct {
		name         string
		cards        string
		expectedHand Hand
		description  string
	}{
		{
			name:         "Straight flush with suit mode",
			cards:        "5h,6h,7h,8h,!9d",
			expectedHand: StraightFlush,
			description:  "Wild 9d in suit mode becomes 9h for straight flush 5-6-7-8-9",
		},
		{
			name:         "Rank mode for four of a kind",
			cards:        "10c,10d,10h,5s,!2s",
			expectedHand: FourOfAKind,
			description:  "Wild in rank mode becomes a 10",
		},
		{
			name:         "Royal flush with suit mode",
			cards:        "14h,13h,12h,11h,!10s",
			expectedHand: RoyalFlush,
			description:  "Wild 10s in suit mode becomes 10h for royal flush",
		},
		{
			name:         "Straight with rank mode",
			cards:        "2c,3d,4h,5s,!9c",
			expectedHand: Straight,
			description:  "Wild 9c in rank mode can fill the 6 or A gap for a wheel or 2-6 straight",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := New(5, deck.CardsFromString(tc.cards))
			assert.Equal(t, tc.expectedHand, h.GetHand(), tc.description)
		})
	}
}

func TestConstrainedVsUnconstrainedComparison(t *testing.T) {
	a := assert.New(t)

	// Test case: Hand that would be a straight flush with unconstrained wilds
	// but not with constrained wilds
	// 5h, 6h, 7h, 8h, !9d
	// Unconstrained: wild could be 9h → straight flush
	// Constrained: wild can be 9 of any suit (suit mode) OR any rank diamond (rank mode)
	//   - Suit mode: 9 of hearts → fills the straight flush!
	//   - Actually this DOES work in constrained mode
	h := New(5, deck.CardsFromString("5h,6h,7h,8h,!9d"))
	sf, ok := h.GetStraightFlush()
	a.True(ok, "Suit-mode wild at rank 9 can become 9h")
	a.Equal(9, sf)

	// Better test: Hand where the wild's rank doesn't match the gap
	// 5h, 6h, 8h, 9h, !3d - need a 7, wild is a 3
	// Unconstrained: wild could be 7h → straight flush
	// Constrained:
	//   - Rank mode: wild can be 7, but stays diamond → no straight flush
	//   - Suit mode: wild can be heart, but stays at rank 3 → no straight flush at 5-6-7-8-9
	h = New(5, deck.CardsFromString("5h,6h,8h,9h,!3d"))
	_, ok = h.GetStraightFlush()
	a.False(ok, "Constrained wild can't make straight flush - wrong rank AND suit")

	// The best this hand can make:
	hand := h.GetHand()
	a.Equal(Flush, hand, "Best hand is a flush (5 hearts via suit mode)")
}

func TestNoWildsUnchanged(t *testing.T) {
	a := assert.New(t)

	// Verify hands without wilds behave exactly the same
	testCases := []struct {
		cards    string
		expected Hand
	}{
		{"14c,13c,12c,11c,10c", RoyalFlush},
		{"9c,8c,7c,6c,5c", StraightFlush},
		{"10c,10d,10h,10s,5c", FourOfAKind},
		{"10c,10d,10h,5s,5c", FullHouse},
		{"2c,4c,6c,8c,10c", Flush},
		{"2c,3d,4h,5s,6c", Straight},
		{"10c,10d,10h,5s,6c", ThreeOfAKind},
		{"10c,10d,5s,5c,6h", TwoPair},
		{"10c,10d,5s,6c,7h", OnePair},
		{"2c,4d,6h,8s,10c", HighCard},
	}

	for _, tc := range testCases {
		t.Run(tc.expected.String(), func(_ *testing.T) {
			h := New(5, deck.CardsFromString(tc.cards))
			a.Equal(tc.expected, h.GetHand())
		})
	}
}

func TestConstrainedWildHelpers(t *testing.T) {
	a := assert.New(t)

	// Test countWildsByMode
	wilds := deck.Hand{
		deck.CardFromString("!5c"),
		deck.CardFromString("!6d"),
		deck.CardFromString("!7h"),
	}
	combos := generateWildCombinations(wilds)

	// All RankMode
	allRank := combos[0] // binary 000
	rankCount, suitCount := countWildsByMode(allRank)
	a.Equal(3, rankCount)
	a.Equal(0, suitCount)

	// Mixed modes - find combo with binary 101 (first and third are SuitMode)
	for _, combo := range combos {
		if len(combo) == 3 && combo[0].Mode == SuitMode && combo[1].Mode == RankMode && combo[2].Mode == SuitMode {
			rankCount, suitCount = countWildsByMode(combo)
			a.Equal(1, rankCount)
			a.Equal(2, suitCount)
			break
		}
	}
}

func TestConstrainedStraightFlushWithMultipleWilds(t *testing.T) {
	a := assert.New(t)

	// Test: 2h, 3h, 4h, !5s, !6d
	// Need both wilds to be hearts for straight flush
	// Wild 5s: suit mode → 5h ✓
	// Wild 6d: suit mode → 6h ✓
	// Result: 2h-3h-4h-5h-6h straight flush
	h := New(5, deck.CardsFromString("2h,3h,4h,!5s,!6d"))
	sf, ok := h.GetStraightFlush()
	a.True(ok, "Both wilds in suit mode can make straight flush")
	a.Equal(6, sf)

	// Test: Ah, Kh, Qh, !Js, !10d
	// Both wilds in suit mode → Royal flush
	h = New(5, deck.CardsFromString("14h,13h,12h,!11s,!10d"))
	a.True(h.GetRoyalFlush(), "Both wilds in suit mode make royal flush")

	// Test: 2c, 3d, 4h, 5s, !6c, !7d
	// For a regular straight, rank-mode wilds work
	// We have 2-3-4-5, need 6 to complete
	// Wild 6c in rank mode can be 6 ✓ (or suit mode keeps it at 6)
	// Wild 7d in rank mode can be... we don't need 7
	// Actually we need A or 6. Wild 6c provides 6, wild 7d provides 7
	// Best straight: 3-4-5-6-7 using both wilds as their natural ranks
	h = New(5, deck.CardsFromString("2c,3d,4h,5s,!6c,!7d"))
	straight, ok := h.GetStraight()
	a.True(ok)
	a.Equal(7, straight, "3-4-5-6-7 straight")
}

func TestConstrainedLowAceStraight(t *testing.T) {
	a := assert.New(t)

	// Test: Ac, 2d, 3h, 4s, !5c (wild 5, not a 5)
	// Wait, if the wild is !5c, it's a wild 5. That's perfect for A-2-3-4-5
	h := New(5, deck.CardsFromString("14c,2d,3h,4s,!5c"))
	straight, ok := h.GetStraight()
	a.True(ok)
	a.Equal(5, straight, "Wheel straight with wild 5")

	// Test: Ac, 2d, 3h, 4s, !9c
	// Wild 9 in rank mode can become a 5 → wheel
	h = New(5, deck.CardsFromString("14c,2d,3h,4s,!9c"))
	straight, ok = h.GetStraight()
	a.True(ok)
	a.Equal(5, straight, "Wheel straight with rank-mode wild becoming 5")

	// Test low-ace straight flush
	// Ah, 2h, 3h, 4h, !5s
	// Wild 5s in suit mode → 5h, completing heart straight flush
	h = New(5, deck.CardsFromString("14h,2h,3h,4h,!5s"))
	sf, ok := h.GetStraightFlush()
	a.True(ok)
	a.Equal(5, sf, "5-high straight flush (wheel)")
}

func TestWildModeConstants(t *testing.T) {
	a := assert.New(t)

	a.Equal(WildMode(0), RankMode)
	a.Equal(WildMode(1), SuitMode)
}

func TestGetRankModeWildsForSuit(t *testing.T) {
	a := assert.New(t)

	assignments := []WildAssignment{
		{Card: deck.CardFromString("!5c"), Mode: RankMode},
		{Card: deck.CardFromString("!6d"), Mode: SuitMode},
		{Card: deck.CardFromString("!7c"), Mode: RankMode},
	}

	// Get rank-mode wilds for clubs
	clubWilds := getRankModeWildsForSuit(assignments, deck.Clubs)
	a.Len(clubWilds, 2)
	a.Equal(5, clubWilds[0].Card.Rank)
	a.Equal(7, clubWilds[1].Card.Rank)

	// Get rank-mode wilds for diamonds (should be empty - the diamond is suit-mode)
	diamondWilds := getRankModeWildsForSuit(assignments, deck.Diamonds)
	a.Len(diamondWilds, 0)
}

func TestGetSuitModeWildsForRank(t *testing.T) {
	a := assert.New(t)

	assignments := []WildAssignment{
		{Card: deck.CardFromString("!5c"), Mode: RankMode},
		{Card: deck.CardFromString("!5d"), Mode: SuitMode},
		{Card: deck.CardFromString("!7c"), Mode: SuitMode},
	}

	// Get suit-mode wilds for rank 5
	rank5Wilds := getSuitModeWildsForRank(assignments, 5)
	a.Len(rank5Wilds, 1)
	a.Equal(deck.Diamonds, rank5Wilds[0].Card.Suit)

	// Get suit-mode wilds for rank 7
	rank7Wilds := getSuitModeWildsForRank(assignments, 7)
	a.Len(rank7Wilds, 1)

	// Get suit-mode wilds for rank 6 (should be empty)
	rank6Wilds := getSuitModeWildsForRank(assignments, 6)
	a.Len(rank6Wilds, 0)
}

func TestConstrainedFlushRankOrdering(t *testing.T) {
	a := assert.New(t)

	// Test that flush ranks are properly ordered
	// 2c, 4c, 6c, 8c, !10d - suit mode wild adds rank 10
	h := New(5, deck.CardsFromString("2c,4c,6c,8c,!10d"))
	flush, ok := h.GetFlush()
	a.True(ok)
	// The flush should be ordered high to low: 10, 8, 6, 4, 2
	a.Equal([]int{10, 8, 6, 4, 2}, flush)

	// Test with rank-mode wild (adds Ace)
	// 2c, 4c, 6c, 8c, !10c - rank mode wild for same suit adds Ace
	h = New(5, deck.CardsFromString("2c,4c,6c,8c,!10c"))
	flush, ok = h.GetFlush()
	a.True(ok)
	// Rank mode wild becomes Ace, so flush should be: 14, 10, 8, 6, 4 (or including 2?)
	// Actually we have 5 clubs already: 2c, 4c, 6c, 8c, 10c
	// The wild is !10c which is a wild 10 of clubs
	// In rank mode, it can be any rank but stays club → Ace of clubs
	// So we have: Ac (wild), 10c, 8c, 6c, 4c = [14, 10, 8, 6, 4]
	a.Contains(flush, 14, "Rank-mode wild becomes Ace")
}

func TestOverall(t *testing.T) {
	a := assert.New(t)

	// cannot change both rank and suit
	h := New(5, deck.CardsFromString("2c,3c,4c,5c,!7d"))
	a.Equal("Flush", h.GetHand().String())
}
