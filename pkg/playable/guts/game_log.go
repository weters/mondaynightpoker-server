package guts

import (
	"mondaynightpoker-server/pkg/deck"
)

// gameLog is the structured record of a game persisted to games.data
// It contains everything required to replay the hand from start to finish.
type gameLog struct {
	Options    Options         `json:"options"`
	InitialPot int             `json:"initialPot"`
	Players    []int64         `json:"players"`
	Rounds     []*gameLogRound `json:"rounds"`
}

// gameLogRound captures one full round (deal → decisions → trades → showdown)
type gameLogRound struct {
	Round     int                `json:"round"`
	Pot       int                `json:"pot"`
	Seats     []*gameLogSeat     `json:"seats"`
	Decisions []*gameLogDecision `json:"decisions"`
	Trades    []*gameLogTrade    `json:"trades,omitempty"`
	DeckHand  []*deck.Card       `json:"deckHand,omitempty"`
	Result    *gameLogShowdown   `json:"result,omitempty"`
}

// gameLogSeat is a player's starting cards in a round
type gameLogSeat struct {
	PlayerID int64        `json:"playerId"`
	Cards    []*deck.Card `json:"cards"`
	Lives    int          `json:"lives,omitempty"`
}

// gameLogDecision records a player's in/out call
type gameLogDecision struct {
	PlayerID int64 `json:"playerId"`
	In       bool  `json:"in"`
}

// gameLogTrade records what a player swapped during the trade phase
type gameLogTrade struct {
	PlayerID     int64        `json:"playerId"`
	DiscardedOut []*deck.Card `json:"discardedOut"`
	NewCards     []*deck.Card `json:"newCards"`
}

// gameLogShowdown is the structured outcome of a single round
type gameLogShowdown struct {
	Winners      []int64        `json:"winners"`
	Losers       []int64        `json:"losers"`
	PlayersIn    []int64        `json:"playersIn"`
	WinningType  string         `json:"winningType,omitempty"`
	PotWon       int            `json:"potWon"`
	PenaltyPaid  int            `json:"penaltyPaid"`
	NextPot      int            `json:"nextPot"`
	AllFolded    bool           `json:"allFolded,omitempty"`
	SingleWinner bool           `json:"singleWinner,omitempty"`
	DeckHand     []*deck.Card   `json:"deckHand,omitempty"`
	DeckWon      bool           `json:"deckWon,omitempty"`
	FinalHands   []*gameLogSeat `json:"finalHands"`
}

// startRound begins a new round entry and snapshots starting hands
func (g *Game) startRound() {
	seats := make([]*gameLogSeat, 0, len(g.participants))
	for _, p := range g.participants {
		cards := make([]*deck.Card, len(p.hand))
		copy(cards, p.hand)
		seats = append(seats, &gameLogSeat{
			PlayerID: p.PlayerID,
			Cards:    cards,
		})
	}
	g.log.Rounds = append(g.log.Rounds, &gameLogRound{
		Round: g.roundNumber,
		Pot:   g.pot,
		Seats: seats,
	})
}

// currentRoundLog returns the in-progress round entry, or nil if none exists
func (g *Game) currentRoundLog() *gameLogRound {
	if len(g.log.Rounds) == 0 {
		return nil
	}
	return g.log.Rounds[len(g.log.Rounds)-1]
}

// recordDecision appends an in/out call to the current round
func (g *Game) recordDecision(playerID int64, in bool) {
	r := g.currentRoundLog()
	if r == nil {
		return
	}
	r.Decisions = append(r.Decisions, &gameLogDecision{
		PlayerID: playerID,
		In:       in,
	})
}

// recordTrade appends a trade entry. discardedOut is what was sent away;
// newCards is what came back (private to the player at the time of the trade).
func (g *Game) recordTrade(playerID int64, discarded, newCards []*deck.Card) {
	r := g.currentRoundLog()
	if r == nil {
		return
	}
	out := make([]*deck.Card, len(discarded))
	copy(out, discarded)
	in := make([]*deck.Card, len(newCards))
	copy(in, newCards)
	r.Trades = append(r.Trades, &gameLogTrade{
		PlayerID:     playerID,
		DiscardedOut: out,
		NewCards:     in,
	})
}

// recordShowdown captures the result of the round plus final hands
func (g *Game) recordShowdown(result *ShowdownResult) {
	r := g.currentRoundLog()
	if r == nil || result == nil {
		return
	}

	winners := make([]int64, len(result.Winners))
	for i, w := range result.Winners {
		winners[i] = w.PlayerID
	}
	losers := make([]int64, len(result.Losers))
	for i, l := range result.Losers {
		losers[i] = l.PlayerID
	}
	playersIn := make([]int64, len(result.PlayersIn))
	for i, p := range result.PlayersIn {
		playersIn[i] = p.PlayerID
	}

	finalHands := make([]*gameLogSeat, 0, len(g.participants))
	for _, p := range g.participants {
		cards := make([]*deck.Card, len(p.hand))
		copy(cards, p.hand)
		finalHands = append(finalHands, &gameLogSeat{
			PlayerID: p.PlayerID,
			Cards:    cards,
		})
	}

	r.Result = &gameLogShowdown{
		Winners:      winners,
		Losers:       losers,
		PlayersIn:    playersIn,
		WinningType:  result.WinningHand.Type.String(),
		PotWon:       result.PotWon,
		PenaltyPaid:  result.PenaltyPaid,
		NextPot:      result.NextPot,
		AllFolded:    result.AllFolded,
		SingleWinner: result.SingleWinner,
		DeckHand:     result.DeckHand,
		DeckWon:      result.DeckWon,
		FinalHands:   finalHands,
	}
}
