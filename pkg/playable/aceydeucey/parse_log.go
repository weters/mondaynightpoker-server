package aceydeucey

import (
	"encoding/json"
	"fmt"
	"strconv"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/gamelog"
)

// persistedRound mirrors the roundJSON written by Round.MarshalJSON.
//
// Unlike every other game, Acey Deucey persists a bare array of rounds rather than
// a wrapper object, so the log carries no game-level ante or options — only the
// rounds themselves.
type persistedRound struct {
	PlayerID int64             `json:"playerId"`
	Games    []persistedSingle `json:"games"`
	Pot      int               `json:"pot"`
}

type persistedSingle struct {
	FirstCard  *deck.Card `json:"firstCard"`
	MiddleCard *deck.Card `json:"middleCard"`
	LastCard   *deck.Card `json:"lastCard"`
	Bet        struct {
		Amount int `json:"amount"`
	} `json:"bet"`
	Result string `json:"result"`
}

// Single-game results that matter to normalization.
const (
	resultWon  = "won"
	resultPass = "pass"
)

// ParseGameLog decodes a persisted Acey Deucey log into a normalized hand.
//
// Acey Deucey is played in turns against the pot rather than against the other
// players: on each turn one player sees two cards, chooses a wager, and a third
// card decides it. That shapes the normalized fields:
//
//   - VoluntarilyPlayed is true when the player wagered anything. Passing is the
//     free option, so declining to bet is the direct analogue of not entering a pot.
//   - Folded is true only when the player passed on every turn they took, since a
//     pass is the closest thing the game has to giving up a hand.
//   - WentToShowdown counts turns that were actually resolved by the third card.
//     There is no opponent to beat, so a resolved wager is the showdown.
//
// AnteCents is always zero: the ante lives in the game options, which this log
// does not persist.
func ParseGameLog(raw json.RawMessage) (*gamelog.Hand, error) {
	var rounds []persistedRound
	if err := json.Unmarshal(raw, &rounds); err != nil {
		return nil, fmt.Errorf("could not decode acey deucey log: %w", err)
	}

	hand := &gamelog.Hand{
		Rounds: len(rounds),
	}

	wagered := make(map[int64]bool)
	turns := make(map[int64]int)

	for i, round := range rounds {
		street := "turn-" + strconv.Itoa(i+1)
		hand.PotCents = round.Pot

		p := hand.Participant(round.PlayerID)
		for _, game := range round.Games {
			turns[round.PlayerID]++

			if game.FirstCard != nil && game.MiddleCard != nil && len(p.StartingCards) == 0 {
				p.StartingCards = []*deck.Card{game.FirstCard, game.MiddleCard}
			}

			if game.Result == resultPass || game.Bet.Amount == 0 {
				hand.AddAction(&gamelog.Action{
					Street:   street,
					PlayerID: round.PlayerID,
					Kind:     gamelog.KindPass,
				})
				continue
			}

			wagered[round.PlayerID] = true
			hand.AddAction(&gamelog.Action{
				Street:      street,
				PlayerID:    round.PlayerID,
				Kind:        gamelog.KindBet,
				AmountCents: game.Bet.Amount,
			})

			p.WentToShowdown = true
			if game.Result == resultWon {
				p.Won = true
			}
			if game.LastCard != nil {
				p.FinalCards = []*deck.Card{game.LastCard}
			}
		}
	}

	for _, p := range hand.Participants {
		p.VoluntarilyPlayed = wagered[p.PlayerID]
		p.Folded = turns[p.PlayerID] > 0 && !wagered[p.PlayerID]
	}

	return hand, nil
}
