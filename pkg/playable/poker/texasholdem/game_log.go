package texasholdem

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
)

// gameLog is the structured record of a game persisted to games.data
// It contains everything required to replay the hand from start to finish.
type gameLog struct {
	Variant      Variant            `json:"variant"`
	Ante         int                `json:"ante"`
	SmallBlind   int                `json:"smallBlind"`
	BigBlind     int                `json:"bigBlind"`
	Seats        []*gameLogSeat     `json:"seats"`
	Streets      []*gameLogStreet   `json:"streets"`
	Actions      []*gameLogAction   `json:"actions"`
	Community    deck.Hand          `json:"community"`
	Pot          int                `json:"pot"`
	Participants []*participantJSON `json:"participants"`
}

// gameLogSeat is the starting state of a player at the table
type gameLogSeat struct {
	PlayerID   int64        `json:"playerId"`
	TableStake int          `json:"tableStake"`
	HoleCards  []*deck.Card `json:"holeCards"`
}

// gameLogStreet records the community cards dealt at a given street
type gameLogStreet struct {
	Street string       `json:"street"`
	Cards  []*deck.Card `json:"cards"`
}

// gameLogAction records a single player action during the hand
type gameLogAction struct {
	Street   string        `json:"street"`
	PlayerID int64         `json:"playerId"`
	Action   action.Action `json:"action"`
	Amount   int           `json:"amount,omitempty"`
	Cards    []*deck.Card  `json:"cards,omitempty"`
	AllIn    bool          `json:"allIn,omitempty"`
}

// captureSeats snapshots each player's hole cards and table stake.
// Call after dealStartingCardsToEachParticipant.
func (g *Game) captureSeats() {
	seats := make([]*gameLogSeat, len(g.participantOrder))
	for i, p := range g.participantOrder {
		hole := make([]*deck.Card, len(p.cards))
		copy(hole, p.cards)
		seats[i] = &gameLogSeat{
			PlayerID:   p.PlayerID,
			TableStake: p.tableStake,
			HoleCards:  hole,
		}
	}
	g.log.Seats = seats
}

// recordStreet records a community-card reveal (flop, turn, river)
func (g *Game) recordStreet(street string, cards ...*deck.Card) {
	streetCards := make([]*deck.Card, len(cards))
	copy(streetCards, cards)
	g.log.Streets = append(g.log.Streets, &gameLogStreet{
		Street: street,
		Cards:  streetCards,
	})
}

// recordAction appends a player action to the log
func (g *Game) recordAction(a action.Action, playerID int64, amount int, cards []*deck.Card, allIn bool) {
	var copied []*deck.Card
	if len(cards) > 0 {
		copied = make([]*deck.Card, len(cards))
		copy(copied, cards)
	}
	g.log.Actions = append(g.log.Actions, &gameLogAction{
		Street:   streetFromDealerState(g.dealerState),
		PlayerID: playerID,
		Action:   a,
		Amount:   amount,
		Cards:    copied,
		AllIn:    allIn,
	})
}

// streetFromDealerState returns a human-readable street name for the current dealer state
func streetFromDealerState(s DealerState) string {
	switch s {
	case DealerStateDiscardRound:
		return "discard"
	case DealerStatePreFlopBettingRound:
		return "preflop"
	case DealerStateFlopBettingRound:
		return "flop"
	case DealerStateTurnBettingRound:
		return "turn"
	case DealerStateFinalBettingRound:
		return "river"
	default:
		return ""
	}
}

// finalGameLog snapshots the final participants, community, and pot
func (g *Game) finalGameLog() *gameLog {
	participants := make([]*participantJSON, len(g.participantOrder))
	for i, pt := range g.participantOrder {
		participants[i] = pt.participantJSON(g, true)
	}

	g.log.Community = g.community
	g.log.Pot = g.potManager.Pots().Total()
	g.log.Participants = participants
	return g.log
}
