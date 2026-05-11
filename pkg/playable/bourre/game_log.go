package bourre

import (
	"mondaynightpoker-server/pkg/deck"
)

// gameLog is the structured record of a Bourré hand persisted to games.data.
// It captures the initial deal, trade-in phase, every trick, and the result so
// the hand can be replayed end-to-end. When a Bourré game requires multiple
// hands (continuation), Parent links to the prior hand's log so the entire
// chain is recoverable from a single root.
type gameLog struct {
	Parent     *gameLog          `json:"parent,omitempty"`
	Options    Options           `json:"options"`
	Ante       int               `json:"ante"`
	InitialPot int               `json:"initialPot"`
	TrumpCard  *deck.Card        `json:"trumpCard,omitempty"`
	Seats      []*gameLogSeat    `json:"seats"`
	Discards   []*gameLogDiscard `json:"discards"`
	Tricks     []*gameLogTrick   `json:"tricks"`
	Result     *Result           `json:"result,omitempty"`
}

// gameLogSeat is a player's starting state
type gameLogSeat struct {
	PlayerID     int64        `json:"playerId"`
	StartingHand []*deck.Card `json:"startingHand"`
}

// gameLogDiscard captures a player's trade-in decision. If `Folded` is true,
// the player chose to fold rather than play this hand.
type gameLogDiscard struct {
	PlayerID  int64        `json:"playerId"`
	Folded    bool         `json:"folded,omitempty"`
	Discarded []*deck.Card `json:"discarded"`
	NewCards  []*deck.Card `json:"newCards"`
}

// gameLogTrick captures one trick: the cards played in order plus the winner.
type gameLogTrick struct {
	Number   int            `json:"number"`
	Plays    []*gameLogPlay `json:"plays"`
	WinnerID int64          `json:"winnerId,omitempty"`
}

// gameLogPlay is a single card played by a single player in a trick.
type gameLogPlay struct {
	PlayerID int64      `json:"playerId"`
	Card     *deck.Card `json:"card"`
}

// captureDeal snapshots each player's 5-card starting hand and the trump card.
func (g *Game) captureDeal() {
	if g.log == nil {
		return
	}
	seats := make([]*gameLogSeat, 0, len(g.playerOrder))
	for player := range g.playerOrder {
		hand := make([]*deck.Card, len(player.hand))
		copy(hand, player.hand)
		seats = append(seats, &gameLogSeat{
			PlayerID:     player.PlayerID,
			StartingHand: hand,
		})
	}
	g.log.Seats = seats
	g.log.TrumpCard = g.trumpCard
}

// recordDiscard captures a player's discard choice. discarded is what they
// threw back (nil if they folded); drawn is the replacement cards they received.
func (g *Game) recordDiscard(playerID int64, folded bool, discarded, drawn []*deck.Card) {
	if g.log == nil {
		return
	}
	dOut := make([]*deck.Card, len(discarded))
	copy(dOut, discarded)
	dIn := make([]*deck.Card, len(drawn))
	copy(dIn, drawn)
	g.log.Discards = append(g.log.Discards, &gameLogDiscard{
		PlayerID:  playerID,
		Folded:    folded,
		Discarded: dOut,
		NewCards:  dIn,
	})
}

// startTrick begins a new trick entry.
func (g *Game) startTrick() {
	if g.log == nil {
		return
	}
	g.log.Tricks = append(g.log.Tricks, &gameLogTrick{
		Number: len(g.log.Tricks) + 1,
		Plays:  make([]*gameLogPlay, 0, len(g.playerOrder)),
	})
}

// recordCardPlay appends a card play to the current trick.
func (g *Game) recordCardPlay(playerID int64, card *deck.Card) {
	if g.log == nil {
		return
	}
	if len(g.log.Tricks) == 0 {
		g.startTrick()
	}
	last := g.log.Tricks[len(g.log.Tricks)-1]
	last.Plays = append(last.Plays, &gameLogPlay{
		PlayerID: playerID,
		Card:     card,
	})
}

// recordTrickWinner sets the winner of the current (most recent) trick.
func (g *Game) recordTrickWinner(playerID int64) {
	if g.log == nil || len(g.log.Tricks) == 0 {
		return
	}
	g.log.Tricks[len(g.log.Tricks)-1].WinnerID = playerID
}

// recordResult attaches the final Result to the log.
func (g *Game) recordResult(r *Result) {
	if g.log == nil {
		return
	}
	g.log.Result = r
}
