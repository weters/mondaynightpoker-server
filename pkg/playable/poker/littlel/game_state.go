package littlel

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker"
	"mondaynightpoker-server/pkg/playable/poker/action"
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
	Name         string             `json:"name"`
	Participants []*participantJSON `json:"participants"`
	DealerID     int64              `json:"dealerId"`
	Round        round              `json:"round"`
	Action       int64              `json:"action"`
	TradeIns     *TradeIns          `json:"tradeIns"`
	InitialDeal  int                `json:"initialDeal"`
	Winners      map[int64]int      `json:"winners"`
}

// State represents the state of the game and the state of the current player
type State struct {
	Participant   *participantJSON `json:"participant"`
	GameState     *GameState       `json:"gameState"`
	PokerState    *poker.State     `json:"pokerState"`
	Actions       []action.Action  `json:"actions"`
	FutureActions []action.Action  `json:"futureActions"`
}
