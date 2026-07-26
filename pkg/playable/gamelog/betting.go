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
	for _, a := range actions {
		kind, ok := a.Action.Kind()
		if !ok {
			continue
		}

		h.AddAction(&Action{
			Street:      a.Street,
			PlayerID:    a.PlayerID,
			Kind:        kind,
			AmountCents: a.Amount,
			Cards:       a.Cards,
			AllIn:       a.AllIn,
		})

		if kind == KindCall || kind.IsAggressive() {
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
