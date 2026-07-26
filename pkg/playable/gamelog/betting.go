package gamelog

import "mondaynightpoker-server/pkg/deck"

// BettingAction is the persisted form of a logged betting action.
//
// Texas Hold'em, Seven Card, and Little L each declare their own gameLogAction
// struct, but all three write the same JSON, so the tags here let those parsers
// decode straight into this type instead of restating the shape. Only Little L
// differs: it records an integer round rather than a named street, so its parser
// embeds this type and supplies the street itself.
type BettingAction struct {
	Street   string       `json:"street"`
	PlayerID int64        `json:"playerId"`
	Action   RawID        `json:"action"`
	Amount   int          `json:"amount"`
	Cards    []*deck.Card `json:"cards"`
	AllIn    bool         `json:"allIn"`
}

// ApplyBettingActions normalizes a hand's betting actions onto h, in order.
//
// A call, bet, or raise marks the player as having voluntarily played: those are
// the actions that put money in by choice. Checking does not qualify, which is
// what keeps a big blind who checks a free preflop out of the VPIP count, and
// blinds and antes never reach here because they are posted by the pot manager
// rather than recorded as actions.
//
// Actions whose identifier is not recognized are skipped rather than treated as an
// error, so a log written by a newer game version still yields usable numbers for
// the actions that are understood.
func (h *Hand) ApplyBettingActions(actions []BettingAction) {
	h.ApplyBettingActionsFunc(actions, nil)
}

// ApplyBettingActionsFunc is ApplyBettingActions with a fallback for action
// identifiers outside the shared betting vocabulary.
//
// Some games interleave non-betting moves into the same action stream as the
// wagering — Seven Card's Chiggs variant lets a player flip a mushroom or play an
// antidote in the middle of a street. Those identifiers are declared in the game
// package, and a game package imports gamelog rather than the other way around, so
// this package cannot resolve them itself. The fallback lets the game's own parser
// supply the mapping without either duplicating the bookkeeping below or losing the
// action's position in the sequence, which is the only record of when it happened
// relative to the betting.
//
// The fallback is consulted only for identifiers RawID.Kind rejects, and an action
// it resolves never sets VoluntarilyPlayed: these are game mechanics rather than
// wagers, and counting one as a voluntary play would make VPIP mean something
// different for the players who happened to be dealt into a variant. An identifier
// that neither RawID.Kind nor the fallback recognizes is skipped, as it is when
// there is no fallback at all. Pass nil for fallback to get exactly
// ApplyBettingActions' behavior.
func (h *Hand) ApplyBettingActionsFunc(actions []BettingAction, fallback func(RawID) (Kind, bool)) {
	for _, a := range actions {
		// betting records whether the identifier resolved through the shared
		// vocabulary, because only those actions can put money in voluntarily.
		kind, betting := a.Action.Kind()
		if !betting {
			if fallback == nil {
				continue
			}

			var ok bool
			if kind, ok = fallback(a.Action); !ok {
				continue
			}
		}

		h.AddAction(&Action{
			Street:      a.Street,
			PlayerID:    a.PlayerID,
			Kind:        kind,
			AmountCents: a.Amount,
			Cards:       a.Cards,
			AllIn:       a.AllIn,
		})

		if betting && (kind == KindCall || kind.IsAggressive()) {
			h.Participant(a.PlayerID).VoluntarilyPlayed = true
		}
	}
}

// ResolveShowdown fills in the outcome fields for a betting game.
//
// folded and winnings are keyed by player id: folded reports whether the player
// gave up the hand, and winnings holds each winner's share of the pot. A player
// reaches a showdown when they did not fold and at least one opponent also did not
// fold — a hand everyone else folded out of is won without one, which is why
// HandsWon and HandsWonAtShowdown are tracked separately.
func (h *Hand) ResolveShowdown(folded map[int64]bool, winnings map[int64]int) {
	contested := 0
	for _, p := range h.Participants {
		if !folded[p.PlayerID] {
			contested++
		}
	}

	for _, p := range h.Participants {
		if folded[p.PlayerID] {
			p.Folded = true
			continue
		}

		p.WentToShowdown = contested > 1
		p.Won = winnings[p.PlayerID] > 0
	}
}
