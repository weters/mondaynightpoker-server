package playable

import (
	"mondaynightpoker-server/pkg/deck"
	"time"
)

// Playable is a game that can be played
type Playable interface {
	// Action performs with a message
	// If playerResponse is not null, that's the response sent directly to the client
	// If updateState is true, it will trigger a state update for all connected clients
	Action(playerID int64, message *PayloadIn) (playerResponse *Response, updateState bool, err error)

	// GetPlayerState returns the current state of the game for the player
	GetPlayerState(playerID int64) (*Response, error)

	// GetEndOfGameDetails returns the details after a game is over
	// If the game is still in progress, nil will be returned and the second param will be false
	GetEndOfGameDetails() (gameOverDetails *GameOverDetails, isGameOver bool)

	// Name returns the name of the game
	Name() string

	// LogChan should return a channel that a game will send log messages to
	LogChan() chan []*LogMessage
}

// LogMessage is the format a game should send log messages in
// If PlayerID is null, assume it's a general statement, otherwise the message will be sent like "{player} did X, Y, Z"
type LogMessage struct {
	UUID      string       `json:"uuid"`
	PlayerIDs []int64      `json:"playerIds"`
	Cards     []*deck.Card `json:"cards"`
	Message   string       `json:"message"`
	Time      time.Time    `json:"time"`
}

// Response is a container to determine who gets the specified message
// If Recipient is 0, it's intended as a broadcast
type Response struct {
	Key     string      `json:"key"`
	Value   string      `json:"value"`
	Data    interface{} `json:"data"`
	Context string      `json:"context"`
}

// OK returns a generic success response
func OK(ctx ...string) *Response {
	res := &Response{
		Key:   "status",
		Value: "OK",
	}

	if len(ctx) == 1 {
		res.Context = ctx[0]
	}

	return res
}

// PayloadIn is the format we expect from the JS client
type PayloadIn struct {
	Action         string                 `json:"action"`
	Subject        string                 `json:"subject"`
	Cards          []*deck.Card           `json:"cards"`
	AdditionalData map[string]interface{} `json:"additionalData"`
	// Context will be passed back on any outgoing message
	Context string `json:"context"`
}

// GameOverDetails provides details on how the game ended
type GameOverDetails struct {
	BalanceAdjustments map[int64]int
	Log                interface{}
}
