package littlel

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"mondaynightpoker-server/pkg/playable/poker/potmanager"
)

type participantJSON struct {
	PlayerID   int64     `json:"playerId"`
	DidFold    bool      `json:"didFold"`
	Balance    int       `json:"balance"`
	CurrentBet int       `json:"currentBet"`
	MinBet     int       `json:"minBet"`
	MaxBet     int       `json:"maxBet"`
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
	Pot          int                `json:"pot"`
	Pots         potmanager.Pots    `json:"pots"`
	Ante         int                `json:"ante"`
	CurrentBet   int                `json:"currentBet"`
	TradeIns     *TradeIns          `json:"tradeIns"`
	InitialDeal  int                `json:"initialDeal"`
	Community    []*deck.Card       `json:"community"`
	Winners      map[int64]int      `json:"winners"`
}

// State represents the state of the game and the state of the current player
type State struct {
	Participant   *participantJSON `json:"participant"`
	GameState     *GameState       `json:"gameState"`
	Actions       []action.Action  `json:"actions"`
	FutureActions []action.Action  `json:"futureActions"`
}
