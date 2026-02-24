package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Bot represents a single bot player connected via WebSocket.
type Bot struct {
	ID        int
	Name      string
	PlayerID  int64
	JWT       string
	AutoPilot bool

	conn      *websocket.Conn
	mu        sync.RWMutex
	gameState *GameState
	sendCh    chan outgoingMessage
	done      chan struct{}
	promptCh  chan<- *Bot // signals the REPL that this bot needs to act
}

type outgoingMessage struct {
	Action         string                 `json:"action"`
	Subject        string                 `json:"subject,omitempty"`
	AdditionalData map[string]interface{} `json:"additionalData,omitempty"`
	Cards          interface{}            `json:"cards,omitempty"`
	Context        string                 `json:"context,omitempty"`
}

type wsResponse struct {
	Key     string          `json:"key"`
	Value   string          `json:"value"`
	Data    json.RawMessage `json:"data"`
	Context string          `json:"context"`
}

// Connect dials the WebSocket and starts read/write goroutines.
func (b *Bot) Connect(serverURL, tableUUID string) error {
	wsURL := strings.Replace(serverURL, "http", "ws", 1)
	u := fmt.Sprintf("%s/table/%s/ws?access_token=%s", wsURL, tableUUID, url.QueryEscape(b.JWT))

	conn, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("bot %s: websocket dial: %w", b.Name, err)
	}

	b.conn = conn
	b.sendCh = make(chan outgoingMessage, 64)
	b.done = make(chan struct{})

	// Handle pings from server
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
	})

	go b.readLoop()
	go b.writeLoop()

	return nil
}

func (b *Bot) readLoop() {
	defer close(b.done)

	for {
		_, message, err := b.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("[%s] read error: %v", b.Name, err)
			}
			return
		}

		var resp wsResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			log.Printf("[%s] unmarshal error: %v", b.Name, err)
			continue
		}

		b.handleMessage(&resp)
	}
}

func (b *Bot) handleMessage(resp *wsResponse) {
	switch resp.Key {
	case "game":
		gs, err := ParseGameState(resp.Value, resp.Data, b.PlayerID)
		if err != nil {
			log.Printf("[%s] parse game state error: %v", b.Name, err)
			return
		}

		b.mu.Lock()
		b.gameState = gs
		b.mu.Unlock()

		if len(gs.ValidActions) > 0 {
			if b.AutoPilot {
				go b.doAutoPilot(gs)
			} else if b.promptCh != nil {
				// Signal REPL non-blocking
				select {
				case b.promptCh <- b:
				default:
				}
			}
		}

	case "gameEnded":
		log.Printf("[%s] game ended", b.Name)

	case "error":
		log.Printf("[%s] server error: %s", b.Name, resp.Value)

	case "clientState":
		// Silently store; not critical for bot

	case "allLogs", "logs":
		// Silently consume log messages

	case "status":
		// OK responses

	case "scheduledGame":
		// Game scheduled notification
	}
}

func (b *Bot) doAutoPilot(gs *GameState) {
	action, ad := AutoPilotAction(gs)
	if action == "" {
		return
	}

	msg := outgoingMessage{
		Action:         action,
		AdditionalData: ad,
	}
	b.Send(msg)
}

func (b *Bot) writeLoop() {
	for {
		select {
		case msg := <-b.sendCh:
			if err := b.conn.WriteJSON(msg); err != nil {
				log.Printf("[%s] write error: %v", b.Name, err)
				return
			}
		case <-b.done:
			return
		}
	}
}

// Send queues a message to be sent via WebSocket.
func (b *Bot) Send(msg outgoingMessage) {
	select {
	case b.sendCh <- msg:
	case <-b.done:
	}
}

// GetGameState returns a snapshot of the current game state.
func (b *Bot) GetGameState() *GameState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.gameState
}

// Close gracefully disconnects the bot.
func (b *Bot) Close() {
	if b.conn != nil {
		_ = b.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		_ = b.conn.Close()
	}
}

// SetActive sends the playerStatus action to mark the bot as active/inactive.
func (b *Bot) SetActive(active bool) {
	b.Send(outgoingMessage{
		Action: "playerStatus",
		AdditionalData: map[string]interface{}{
			"active": active,
		},
	})
}

// StartGame sends a createGame action to start a game.
func (b *Bot) StartGame(gameName string) {
	b.Send(outgoingMessage{
		Action:  "createGame",
		Subject: gameName,
	})
}
