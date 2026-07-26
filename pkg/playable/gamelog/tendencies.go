package gamelog

// Tendencies is one player's behavioral profile aggregated over many hands.
//
// It answers how someone plays rather than how much they won: the ledger already
// records the money. Every field is a raw count so callers can derive whatever
// rates they want and, more importantly, so a caller can tell the difference
// between "folds 100% of the time" over three hands and over three hundred.
type Tendencies struct {
	// HandsPlayed is the number of hands the player appeared in.
	HandsPlayed int
	// HandsVoluntarilyPlayed is the number of hands where the player chose to
	// commit beyond any forced ante or blind. This is the cross-game VPIP.
	HandsVoluntarilyPlayed int
	// HandsFolded is the number of hands the player folded or dropped out of.
	HandsFolded int
	// HandsToShowdown is the number of hands the player was still contesting when
	// the hand resolved.
	HandsToShowdown int
	// HandsWonAtShowdown is the subset of HandsToShowdown the player won.
	HandsWonAtShowdown int
	// HandsWon is every hand the player won, including those won without a
	// showdown because everyone else folded.
	HandsWon int

	// Action tallies across every hand.
	Bets     int
	Raises   int
	Calls    int
	Checks   int
	Folds    int
	Discards int
	Trades   int

	// AmountWageredCents is the total the player put into pots through their own
	// actions, excluding forced antes.
	AmountWageredCents int
}

// Add folds a single hand's participation record into the aggregate.
func (t *Tendencies) Add(p *Participation) {
	if p == nil {
		return
	}

	t.HandsPlayed++
	if p.VoluntarilyPlayed {
		t.HandsVoluntarilyPlayed++
	}
	if p.Folded {
		t.HandsFolded++
	}
	if p.WentToShowdown {
		t.HandsToShowdown++
		if p.Won {
			t.HandsWonAtShowdown++
		}
	}
	if p.Won {
		t.HandsWon++
	}

	t.AmountWageredCents += p.AmountWageredCents

	t.Bets += p.Counts[KindBet]
	t.Raises += p.Counts[KindRaise]
	t.Calls += p.Counts[KindCall]
	t.Checks += p.Counts[KindCheck]
	t.Folds += p.Counts[KindFold] + p.Counts[KindDropOut]
	t.Discards += p.Counts[KindDiscard]
	t.Trades += p.Counts[KindTrade]
}

// AggressionFactor is the ratio of aggressive actions (bets and raises) to calls,
// the standard measure of whether a player drives betting or follows it.
//
// It is undefined when the player has never called: dividing by zero would report
// an infinitely aggressive player on the strength of a single bet. The second
// return value is false in that case, and callers should omit the statistic rather
// than substitute a number.
func (t *Tendencies) AggressionFactor() (float64, bool) {
	if t.Calls == 0 {
		return 0, false
	}

	return float64(t.Bets+t.Raises) / float64(t.Calls), true
}

// Rate divides count by HandsPlayed, reporting false when no hands were played.
// Callers use it for the participation rates (VPIP, fold rate, showdown rate) so
// the zero-hands case is handled the same way everywhere.
func (t *Tendencies) Rate(count int) (float64, bool) {
	if t.HandsPlayed == 0 {
		return 0, false
	}

	return float64(count) / float64(t.HandsPlayed), true
}

// ShowdownWinRate is the share of contested showdowns the player won, reporting
// false when they never reached one. It is deliberately not a Rate over
// HandsPlayed: folding more does not make someone better at showdowns.
func (t *Tendencies) ShowdownWinRate() (float64, bool) {
	if t.HandsToShowdown == 0 {
		return 0, false
	}

	return float64(t.HandsWonAtShowdown) / float64(t.HandsToShowdown), true
}
