// Package gamelog defines a normalized, game-agnostic view of the structured log
// each game persists to the games.data jsonb column.
//
// Every game type writes its own log shape (see the game_log.go file in each game
// package), and those shapes have nothing in common: Bourré records tricks and a
// trump card, Guts records per-round in/out decisions, and the poker variants
// record streets and betting actions. That diversity is correct for replay, but it
// makes cross-game analysis impossible without a common denominator.
//
// A Hand is that denominator. Each game package provides a ParseGameLog function
// that decodes its own persisted JSON and projects it into a Hand, so the analytics
// layer can aggregate across every game type without knowing any of them. The
// projection is lossy on purpose: it keeps who played, what they chose to do, and
// how the hand resolved, and it drops the game-specific mechanics that cannot be
// compared across game types.
//
// Money is in integer cents throughout, matching the rest of the codebase.
package gamelog

import (
	"encoding/json"
	"fmt"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
)

// Kind is a normalized player action.
//
// The betting kinds map one-to-one onto poker/action.Action. The remaining kinds
// exist so games without a betting round still describe their decisions in the
// same vocabulary: Guts players declare in or out, Bourré players fold or play
// cards to tricks, and Pass the Poop players stay or trade.
type Kind string

// Normalized action kinds.
const (
	// KindFold is a player surrendering their claim on the pot. Bourré's
	// fold-at-trade-in and Guts' "out" declaration both normalize to KindDropOut
	// rather than KindFold, because neither forfeits money already wagered.
	KindFold  Kind = "fold"
	KindCheck Kind = "check"
	KindCall  Kind = "call"
	KindBet   Kind = "bet"
	KindRaise Kind = "raise"

	// KindDiscard and KindTrade are card-exchange actions.
	KindDiscard Kind = "discard"
	KindTrade   Kind = "trade"

	// KindStayIn and KindDropOut are commitment decisions in games that have no
	// betting round: the player either commits to contest the pot or steps aside.
	KindStayIn  Kind = "stay-in"
	KindDropOut Kind = "drop-out"

	// KindPlayCard is a card played to a trick (Bourré).
	KindPlayCard Kind = "play-card"

	// KindPass is declining to act when passing is a distinct choice from checking
	// (Acey Deucey's pass, which forfeits the turn without a wager).
	KindPass Kind = "pass"
)

// IsAggressive reports whether the action puts money in voluntarily and applies
// pressure: a bet or a raise. It backs the aggression factor, which is the ratio
// of aggressive actions to calls.
func (k Kind) IsAggressive() bool {
	return k == KindBet || k == KindRaise
}

// Action is a single normalized player action within a hand.
type Action struct {
	// Sequence is the action's zero-based position in the hand, assigned by the
	// parser in the order the actions were recorded.
	Sequence int `json:"sequence"`
	// Street is the phase of the hand the action occurred in. The vocabulary is
	// game-specific ("preflop"/"flop" for hold'em, "third-street" for seven card,
	// "round-1" for the round-based games); it is descriptive, not comparable
	// across game types.
	Street      string       `json:"street,omitempty"`
	PlayerID    int64        `json:"playerId"`
	Kind        Kind         `json:"kind"`
	AmountCents int          `json:"amountCents,omitempty"`
	Cards       []*deck.Card `json:"cards,omitempty"`
	AllIn       bool         `json:"allIn,omitempty"`
}

// Participation is one player's involvement in one hand.
//
// VoluntarilyPlayed is the cross-game analogue of VPIP: it is true when the player
// made a choice that committed them to contest the pot beyond whatever the game
// forced them to post. Each parser decides what qualifies, because only it knows
// which contributions were forced (antes and blinds never count).
type Participation struct {
	PlayerID      int64        `json:"playerId"`
	StartingCards []*deck.Card `json:"startingCards,omitempty"`
	FinalCards    []*deck.Card `json:"finalCards,omitempty"`

	VoluntarilyPlayed bool `json:"voluntarilyPlayed"`
	Folded            bool `json:"folded"`
	WentToShowdown    bool `json:"wentToShowdown"`
	Won               bool `json:"won"`

	// AmountWageredCents is what the player put into the pot through their own
	// actions during the hand. It excludes the forced ante and is not the same as
	// their net result, which comes from the ledger rather than the log.
	AmountWageredCents int `json:"amountWageredCents"`

	// Counts tallies the player's actions by kind.
	Counts map[Kind]int `json:"counts,omitempty"`
}

// count records one action of the given kind.
func (p *Participation) count(k Kind) {
	if p.Counts == nil {
		p.Counts = make(map[Kind]int)
	}
	p.Counts[k]++
}

// Hand is the normalized summary of one completed game.
type Hand struct {
	// GameType is the games.game_type identifier (e.g. "texas-hold-em").
	GameType string `json:"gameType"`
	// Variant distinguishes sub-games within a game type (e.g. "pineapple" for
	// hold'em, "baseball" for seven card). It is empty when the game type has no
	// variants.
	Variant string `json:"variant,omitempty"`

	AnteCents int `json:"anteCents"`
	PotCents  int `json:"potCents"`
	// Rounds is the number of distinct rounds or hands the game ran. Games that
	// resolve in a single pass report 1.
	Rounds int `json:"rounds"`
	// Board is the shared/community cards, if the game has any.
	Board []*deck.Card `json:"board,omitempty"`

	Participants []*Participation `json:"participants"`
	Actions      []*Action        `json:"actions"`
}

// FindParticipant returns the participation record for the player, or nil when the
// hand does not name them. Use it to ask whether a player took part; use
// Participant to accumulate their involvement.
func (h *Hand) FindParticipant(playerID int64) *Participation {
	for _, p := range h.Participants {
		if p.PlayerID == playerID {
			return p
		}
	}

	return nil
}

// Participant returns the participation record for the player, creating it if the
// hand does not have one yet. Parsers use it to accumulate a player's involvement
// without tracking insertion order themselves.
func (h *Hand) Participant(playerID int64) *Participation {
	if p := h.FindParticipant(playerID); p != nil {
		return p
	}

	p := &Participation{PlayerID: playerID}
	h.Participants = append(h.Participants, p)
	return p
}

// AddAction appends a normalized action to the hand and reflects it in the acting
// player's participation record: the action is counted, any wagered amount is
// accumulated, and a fold is recorded.
//
// It deliberately does not infer VoluntarilyPlayed. Whether an action was
// voluntary depends on game rules the Hand does not model (a hold'em big blind
// posts a bet without choosing to), so each parser sets that flag itself.
func (h *Hand) AddAction(a *Action) {
	a.Sequence = len(h.Actions)
	h.Actions = append(h.Actions, a)

	p := h.Participant(a.PlayerID)
	p.count(a.Kind)
	p.AmountWageredCents += a.AmountCents

	switch a.Kind {
	case KindFold, KindDropOut:
		p.Folded = true
	}
}

// RawID decodes an identifier from the persisted log that may be stored either as
// a bare JSON string or as an object carrying an "id" field.
//
// Several game packages define MarshalJSON on their enum types to emit
// {"id":"fold","name":"Fold"} but never define a matching UnmarshalJSON, so those
// values do not round-trip back into their original Go type. Actions and Texas
// Hold'em variants are both stored that way. RawID accepts either form, so parsing
// works regardless of which encoding produced the log.
type RawID string

// UnmarshalJSON implements json.Unmarshaler, accepting either a bare string or an
// {"id": ...} object.
func (r *RawID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*r = RawID(s)
		return nil
	}

	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return fmt.Errorf("could not decode identifier: %w", err)
	}

	*r = RawID(obj.ID)
	return nil
}

// Kind maps the persisted action identifier onto a normalized Kind. Unrecognized
// identifiers return false so a parser can decide whether to skip the action or
// fail; a log written by a newer game version should not break analysis of the
// actions that are understood.
//
// The cases are the poker/action constants rather than string literals, so
// renaming one there is a compile error here instead of a silently unrecognized
// action. Seven Card declares its own action type with the same identifiers and
// cannot be referenced from this package (it imports gamelog), but its parser
// feeds the same strings through here.
func (r RawID) Kind() (Kind, bool) {
	switch action.Action(r) {
	case action.Fold:
		return KindFold, true
	case action.Check:
		return KindCheck, true
	case action.Call:
		return KindCall, true
	case action.Bet:
		return KindBet, true
	case action.Raise:
		return KindRaise, true
	case action.Discard:
		return KindDiscard, true
	case action.Trade:
		return KindTrade, true
	}

	return "", false
}
