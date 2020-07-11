package littlel

import (
	"mondaynightpoker-server/pkg/deck"
)

type participantJSON struct {
	PlayerID   int64     `json:"playerId"`
	DidFold    bool      `json:"didFold"`
	Balance    int       `json:"balance"`
	CurrentBet int       `json:"currentBet"`
	Traded     int       `json:"traded"`
	Hand       deck.Hand `json:"hand"`
	HandRank   string    `json:"handRank"`
}

// GameState is the state of the game
type GameState struct {
	Participants []*participantJSON `json:"participants"`
	DealerID     int64              `json:"dealerId"`
	Stage        stage              `json:"stage"`
	Action       int64              `json:"action"`
	Pot          int                `json:"pot"`
	Ante         int                `json:"ante"`
	CurrentBet   int                `json:"currentBet"`
	MaxBet       int                `json:"maxBet"`
	TradeIns     TradeIns           `json:"tradeIns"`
	InitialDeal  int                `json:"initialDeal"`
	Community    []*deck.Card       `json:"community"`
	Winners      []int64            `json:"winners"`
}

// State represents the state of the game and the state of the current player
type State struct {
	Participant *participantJSON `json:"participant"`
	GameState   *GameState       `json:"gameState"`
	Actions     []Action         `json:"actions"`
}
