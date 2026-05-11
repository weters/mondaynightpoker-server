package littlel

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
)

// gameLog is the structured record of a game persisted to games.data
// It contains everything required to replay the hand from start to finish.
type gameLog struct {
	Ante         int                `json:"ante"`
	InitialDeal  int                `json:"initialDeal"`
	TradeIns     []int              `json:"tradeIns"`
	Seats        []*gameLogSeat     `json:"seats"`
	Community    []*deck.Card       `json:"community"`
	Actions      []*gameLogAction   `json:"actions"`
	Participants []*participantJSON `json:"participants"`
	Winners      map[int64]int      `json:"winners"`
}

// gameLogSeat is the starting state of a player
type gameLogSeat struct {
	PlayerID     int64        `json:"playerId"`
	TableStake   int          `json:"tableStake"`
	StartingHand []*deck.Card `json:"startingHand"`
}

// gameLogAction records a single player action during the hand
type gameLogAction struct {
	Round    round         `json:"round"`
	PlayerID int64         `json:"playerId"`
	Action   action.Action `json:"action"`
	Amount   int           `json:"amount,omitempty"`
	Cards    []*deck.Card  `json:"cards,omitempty"`
	AllIn    bool          `json:"allIn,omitempty"`
}

// captureSeats snapshots each player's dealt hand. Call after DealCards.
func (g *Game) captureSeats() {
	seats := make([]*gameLogSeat, 0, len(g.playerIDs))
	for _, id := range g.playerIDs {
		p := g.idToParticipant[id]
		hand := make([]*deck.Card, len(p.hand))
		copy(hand, p.hand)
		seats = append(seats, &gameLogSeat{
			PlayerID:     id,
			TableStake:   p.tableStake,
			StartingHand: hand,
		})
	}
	g.log.Seats = seats
}

// recordAction appends a player action to the log
func (g *Game) recordAction(a action.Action, playerID int64, amount int, cards []*deck.Card, allIn bool) {
	var copied []*deck.Card
	if len(cards) > 0 {
		copied = make([]*deck.Card, len(cards))
		copy(copied, cards)
	}
	g.log.Actions = append(g.log.Actions, &gameLogAction{
		Round:    g.round,
		PlayerID: playerID,
		Action:   a,
		Amount:   amount,
		Cards:    copied,
		AllIn:    allIn,
	})
}

// finalGameLog snapshots community + winners + final hands and returns the log
func (g *Game) finalGameLog() *gameLog {
	g.log.Community = g.community
	if g.winners != nil {
		winners := make(map[int64]int)
		for p, amt := range g.winners {
			winners[p.PlayerID] = amt
		}
		g.log.Winners = winners
	}

	participants := make([]*participantJSON, 0, len(g.playerIDs))
	for _, id := range g.playerIDs {
		p := g.idToParticipant[id]
		pJSON := &participantJSON{
			PlayerID:   p.PlayerID,
			DidFold:    p.didFold,
			Balance:    p.balance,
			CurrentBet: 0,
			Traded:     p.traded,
			Hand:       p.hand,
		}
		if !p.didFold {
			pJSON.HandRank = p.GetBestHand(g.GetCommunityCards()).analyzer.GetHand().String()
		}
		participants = append(participants, pJSON)
	}
	g.log.Participants = participants
	return g.log
}
