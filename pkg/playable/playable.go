package playable

import (
	"fmt"
	"github.com/google/uuid"
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
	LogChan() <-chan []*LogMessage
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
	Action         string         `json:"action"`
	Subject        string         `json:"subject"`
	Cards          []*deck.Card   `json:"cards"`
	AdditionalData AdditionalData `json:"additionalData"`
	// Context will be passed back on any outgoing message
	Context string `json:"context"`
}

// GameOverDetails provides details on how the game ended
type GameOverDetails struct {
	BalanceAdjustments map[int64]int
	Log                interface{}
}

// AdditionalData provides additional data in a payload
type AdditionalData map[string]interface{}

// GetString returns a string for the given key
func (a AdditionalData) GetString(key string) (string, bool) {
	s, ok := a[key].(string)
	return s, ok
}

// GetInt returns an integer value for the given key
func (a AdditionalData) GetInt(key string) (int, bool) {
	floatVal, ok := a[key].(float64)
	if !ok {
		return 0, false
	}

	return int(floatVal), true
}

// GetBool returns a boolean value for the given key
func (a AdditionalData) GetBool(key string) (bool, bool) {
	boolVal, ok := a[key].(bool)
	if !ok {
		return false, false
	}

	return boolVal, true
}

// GetIntSlice returns a slice of integers
func (a AdditionalData) GetIntSlice(key string) ([]int, bool) {
	switch slice := a[key].(type) {
	case []float64:
		ints := make([]int, len(slice))
		for i, val := range slice {
			ints[i] = int(val)
		}
		return ints, true
	case []interface{}:
		ints := make([]int, len(slice))
		for i, val := range slice {
			floatVal, ok := val.(float64)
			if !ok {
				return nil, false
			}

			ints[i] = int(floatVal)
		}
		return ints, true
	}

	return nil, false
}

// SimpleLogMessage returns a new LogMessage
func SimpleLogMessage(playerID int64, format string, a ...interface{}) *LogMessage {
	var playerIDs []int64
	if playerID > 0 {
		playerIDs = []int64{playerID}
	}

	return &LogMessage{
		UUID:      uuid.New().String(),
		PlayerIDs: playerIDs,
		Message:   fmt.Sprintf(format, a...),
		Time:      time.Now(),
	}
}

// SimpleLogMessageSlice returns a single log message
func SimpleLogMessageSlice(playerID int64, format string, a ...interface{}) []*LogMessage {
	return []*LogMessage{SimpleLogMessage(playerID, format, a...)}
}
