package guts

import (
	"encoding/json"
	"fmt"
	"strconv"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"
)

// persistedLog mirrors the JSON written by gameLog.
//
// The options field has no struct tags on the live type, so it serializes with Go
// field names rather than lowercased JSON ones — hence the capitalized tag below.
type persistedLog struct {
	Options struct {
		Ante int `json:"Ante"`
	} `json:"options"`
	Players []int64          `json:"players"`
	Rounds  []persistedRound `json:"rounds"`
}

type persistedRound struct {
	Round     int                 `json:"round"`
	Pot       int                 `json:"pot"`
	Seats     []persistedSeat     `json:"seats"`
	Decisions []persistedDecision `json:"decisions"`
	Trades    []persistedTrade    `json:"trades"`
	Result    *persistedShowdown  `json:"result"`
}

type persistedSeat struct {
	PlayerID int64        `json:"playerId"`
	Cards    []*deck.Card `json:"cards"`
}

type persistedDecision struct {
	PlayerID int64 `json:"playerId"`
	In       bool  `json:"in"`
}

type persistedTrade struct {
	PlayerID     int64        `json:"playerId"`
	DiscardedOut []*deck.Card `json:"discardedOut"`
}

type persistedShowdown struct {
	Winners    []int64         `json:"winners"`
	PlayersIn  []int64         `json:"playersIn"`
	FinalHands []persistedSeat `json:"finalHands"`
}

// ParseGameLog decodes a persisted Guts log into a normalized hand.
//
// A Guts game runs one or more rounds, and the normalized hand covers the whole
// game rather than a single round. A player counts as having voluntarily played if
// they declared in on any round, and as having folded only if they declared out on
// every round they were dealt into — staying out of one round of a five-round game
// is not the same as folding the game.
//
// Starting cards are taken from the first round and final cards from the last
// showdown, so a multi-round game reports the hand the player began with and the
// hand they finished on.
func ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error) {
	var log persistedLog
	if err := json.Unmarshal(raw, &log); err != nil {
		return nil, fmt.Errorf("could not decode guts log: %w", err)
	}

	hand := &gamelog.Hand{
		AnteCents: log.Options.Ante,
		Rounds:    len(log.Rounds),
	}

	for _, id := range log.Players {
		hand.Participant(id)
	}

	for i, round := range log.Rounds {
		street := "round-" + strconv.Itoa(round.Round)
		hand.PotCents = round.Pot

		if i == 0 {
			for _, seat := range round.Seats {
				hand.Participant(seat.PlayerID).StartingCards = seat.Cards
			}
		}

		for _, d := range round.Decisions {
			kind := gamelog.KindDropOut
			if d.In {
				kind = gamelog.KindStayIn
			}

			hand.AddAction(&gamelog.Action{
				Street:   street,
				PlayerID: d.PlayerID,
				Kind:     kind,
			})
		}

		for _, t := range round.Trades {
			hand.AddAction(&gamelog.Action{
				Street:   street,
				PlayerID: t.PlayerID,
				Kind:     gamelog.KindTrade,
				Cards:    t.DiscardedOut,
			})
		}

		applyShowdown(hand, round.Result)
	}

	// A stay-in action is recorded exactly when a player declared in, so the tally
	// already answers this and no parallel map is needed. AddAction marks a player
	// folded on any drop-out; only a player who never once declared in actually sat
	// the whole game out.
	for _, p := range hand.Participants {
		declaredIn := p.Counts[gamelog.KindStayIn] > 0
		p.VoluntarilyPlayed = declaredIn
		p.Folded = !declaredIn
	}

	return hand, nil
}

// applyShowdown folds one round's result into the hand's participation records.
// Reaching a showdown requires more than one player to have declared in;
// a lone player takes the pot without contest.
func applyShowdown(hand *gamelog.Hand, result *persistedShowdown) {
	if result == nil {
		return
	}

	if len(result.PlayersIn) > 1 {
		for _, id := range result.PlayersIn {
			hand.Participant(id).WentToShowdown = true
		}
	}

	for _, id := range result.Winners {
		hand.Participant(id).Won = true
	}

	for _, seat := range result.FinalHands {
		hand.Participant(seat.PlayerID).FinalCards = seat.Cards
	}
}
