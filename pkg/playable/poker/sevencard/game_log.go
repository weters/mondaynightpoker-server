package sevencard

import (
	"mondaynightpoker-server/pkg/deck"
)

// gameLog is the structured record of a game persisted to games.data
// It contains everything required to replay the hand from start to finish.
type gameLog struct {
	Variant    string           `json:"variant"`
	Ante       int              `json:"ante"`
	Seats      []*gameLogSeat   `json:"seats"`
	Deals      []*gameLogDeal   `json:"deals"`
	Actions    []*gameLogAction `json:"actions"`
	FinalState GameState        `json:"finalState"`
}

// gameLogSeat is the starting state of a player
type gameLogSeat struct {
	PlayerID   int64 `json:"playerId"`
	TableStake int   `json:"tableStake"`
	SeatIndex  int   `json:"seatIndex"`
}

// gameLogDealCard records a single card dealt to a player at a given street
type gameLogDealCard struct {
	PlayerID int64      `json:"playerId"`
	Card     *deck.Card `json:"card"`
	FaceUp   bool       `json:"faceUp"`
}

// gameLogDeal is a deal phase (initial / 4th / 5th / 6th / river)
type gameLogDeal struct {
	Street string             `json:"street"`
	Cards  []*gameLogDealCard `json:"cards"`
}

// gameLogAction records a single player action during the hand
type gameLogAction struct {
	Street   string       `json:"street"`
	PlayerID int64        `json:"playerId"`
	Action   Action       `json:"action"`
	Amount   int          `json:"amount,omitempty"`
	Cards    []*deck.Card `json:"cards,omitempty"`
	AllIn    bool         `json:"allIn,omitempty"`
}

// streetName returns a human-readable name for the current round
func streetName(r round) string {
	switch r {
	case beforeDeal:
		return "initial"
	case firstBettingRound:
		return "third-street"
	case secondBettingRound:
		return "fourth-street"
	case thirdBettingRound:
		return "fifth-street"
	case fourthBettingRound:
		return "sixth-street"
	case finalBettingRound:
		return "river"
	case revealWinner:
		return "showdown"
	default:
		return ""
	}
}

// recordDeal records a deal of cards to participants at the start of a street
func (g *Game) recordDeal(street string, dealt []*gameLogDealCard) {
	g.log.Deals = append(g.log.Deals, &gameLogDeal{
		Street: street,
		Cards:  dealt,
	})
}

// recordAction appends a player action to the log
func (g *Game) recordAction(a Action, playerID int64, amount int, cards []*deck.Card, allIn bool) {
	var copied []*deck.Card
	if len(cards) > 0 {
		copied = make([]*deck.Card, len(cards))
		copy(copied, cards)
	}
	g.log.Actions = append(g.log.Actions, &gameLogAction{
		Street:   streetName(g.round),
		PlayerID: playerID,
		Action:   a,
		Amount:   amount,
		Cards:    copied,
		AllIn:    allIn,
	})
}

// finalGameLog captures the final state and returns the persisted log
func (g *Game) finalGameLog() *gameLog {
	g.log.FinalState = g.getGameState()
	return g.log
}
