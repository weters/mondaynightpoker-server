package littlel

import (
	"mondaynightpoker-server/pkg/deck"
)

type participantJSON struct {
	PlayerID   int64     `json:"playerId"`
	DidFold    bool      `json:"didFold"`
	Balance    int       `json:"balance"`
	CurrentBet int       `json:"currentBet"`
	Hand       deck.Hand `json:"hand"`
	HandRank   string    `json:"handRank"`
}

// GameState is the state of the game
type GameState struct {
	Participants []*participantJSON `json:"participants"`
	Stage        stage              `json:"stage"`
	Action       int64              `json:"action"`
	Pot          int                `json:"pot"`
	CurrentBet   int                `json:"currentBet"`
	TradeIns     TradeIns           `json:"tradeIns"`
	InitialDeal  int                `json:"initialDeal"`
	Community    []*deck.Card       `json:"community"`
}

// State represents the state of the game and the state of the current player
type State struct {
	Participant *participantJSON `json:"participant"`
	GameState   *GameState       `json:"gameState"`
	Actions     []Action         `json:"actions"`
}
