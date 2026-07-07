package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

// reconnectInterval is the pause between reconnect attempts after a dropped connection.
const reconnectInterval = 2 * time.Second

// Bot represents a single bot player connected via WebSocket.
type Bot struct {
	ID        int
	Name      string
	PlayerID  int64
	JWT       string
	AutoPilot bool

	serverURL     string
	tableUUID     string
	conn          *websocket.Conn
	mu            sync.RWMutex
	gameState     *GameState
	autoPilotBusy bool // true while an autopilot goroutine is running
	disconnected  bool // true after a dropped connection, until reconnected
	sendCh        chan outgoingMessage
	closed        chan struct{}
	closeOnce     sync.Once
	program       *tea.Program // TUI program for sending messages
	forwardLogs   bool         // only one bot should forward logs/clientState to avoid duplicates
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

// Connect dials the WebSocket and starts the connection manager, which
// reconnects automatically if the connection drops.
func (b *Bot) Connect(serverURL, tableUUID string) error {
	b.serverURL = serverURL
	b.tableUUID = tableUUID
	b.sendCh = make(chan outgoingMessage, 64)
	b.closed = make(chan struct{})

	conn, err := b.dial()
	if err != nil {
		return err
	}

	b.setConn(conn, false)
	go b.run(conn)

	return nil
}

func (b *Bot) dial() (*websocket.Conn, error) {
	wsURL := strings.Replace(b.serverURL, "http", "ws", 1)
	u := fmt.Sprintf("%s/table/%s/ws?access_token=%s", wsURL, b.tableUUID, url.QueryEscape(b.JWT))

	conn, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("bot %s: websocket dial: %w", b.Name, err)
	}

	return conn, nil
}

func (b *Bot) setConn(conn *websocket.Conn, disconnected bool) {
	b.mu.Lock()
	b.conn = conn
	b.disconnected = disconnected
	b.mu.Unlock()
}

// Disconnected reports whether the bot lost its connection and has not yet
// reconnected.
func (b *Bot) Disconnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.disconnected
}

// run serves the current connection and reconnects on failure until Close.
func (b *Bot) run(conn *websocket.Conn) {
	for {
		b.serveConn(conn)

		select {
		case <-b.closed:
			return
		default:
		}

		b.setConn(nil, true)
		b.sendTUI(BotConnMsg{BotID: b.ID, Connected: false})

		var err error
		for {
			select {
			case <-b.closed:
				return
			case <-time.After(reconnectInterval):
			}

			if conn, err = b.dial(); err == nil {
				break
			}
		}

		b.setConn(conn, false)
		b.sendTUI(BotConnMsg{BotID: b.ID, Connected: true})
		b.SetActive(true)
	}
}

// serveConn reads and writes on a single connection until it fails.
func (b *Bot) serveConn(conn *websocket.Conn) {
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
	})

	done := make(chan struct{})
	go b.writeLoop(conn, done)
	b.readLoop(conn)
	close(done)
}

func (b *Bot) readLoop(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
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
			b.sendTUI(ErrorMsg{BotID: b.ID, Message: fmt.Sprintf("parse game state: %v", err)})
			return
		}

		b.mu.Lock()
		b.gameState = gs
		shouldAutoPilot := b.AutoPilot && len(gs.ValidActions) > 0 && !b.autoPilotBusy
		if shouldAutoPilot {
			b.autoPilotBusy = true
		}
		b.mu.Unlock()

		// Notify TUI of state change
		b.sendTUI(BotStateMsg{BotID: b.ID})

		if len(gs.ValidActions) > 0 && shouldAutoPilot {
			go b.doAutoPilot(gs)
		}

	case "gameEnded":
		b.sendTUI(GameEndedMsg{BotID: b.ID})

	case "error":
		b.sendTUI(ErrorMsg{BotID: b.ID, Message: resp.Value})

	case "clientState":
		if b.forwardLogs {
			names := parseClientState(resp.Data)
			if len(names) > 0 {
				b.sendTUI(ClientStateMsg{PlayerNames: names})
			}
		}

	case "allLogs", "logs":
		if b.forwardLogs {
			entries := parseLogs(resp.Data)
			for _, entry := range entries {
				b.sendTUI(GameLogMsg(entry))
			}
		}

	case "status":
		// OK responses

	case "scheduledGame":
		// Game scheduled notification
	}
}

// sendTUI sends a message to the TUI program if one is set.
func (b *Bot) sendTUI(msg tea.Msg) {
	if b.program != nil {
		b.program.Send(msg)
	}
}

func (b *Bot) doAutoPilot(gs *GameState) {
	defer func() {
		b.mu.Lock()
		b.autoPilotBusy = false
		b.mu.Unlock()
	}()

	msg := AutoPilotAction(gs)
	if msg == nil {
		return
	}

	b.Send(*msg)
}

func (b *Bot) writeLoop(conn *websocket.Conn, done chan struct{}) {
	for {
		select {
		case msg := <-b.sendCh:
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("[%s] write error: %v", b.Name, err)
				return
			}
		case <-done:
			return
		case <-b.closed:
			return
		}
	}
}

// Send queues a message to be sent via WebSocket.
func (b *Bot) Send(msg outgoingMessage) {
	select {
	case b.sendCh <- msg:
	case <-b.closed:
	}
}

// GetGameState returns a snapshot of the current game state.
func (b *Bot) GetGameState() *GameState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.gameState
}

// Close gracefully disconnects the bot and stops reconnect attempts.
func (b *Bot) Close() {
	b.closeOnce.Do(func() {
		if b.closed != nil {
			close(b.closed)
		}
	})

	b.mu.RLock()
	conn := b.conn
	b.mu.RUnlock()

	if conn != nil {
		_ = conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		_ = conn.Close()
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

// TerminateGame ends the current game early.
func (b *Bot) TerminateGame() {
	b.Send(outgoingMessage{Action: "terminateGame"})
}

// CancelGame cancels a scheduled (pending) game.
func (b *Bot) CancelGame() {
	b.Send(outgoingMessage{Action: "cancelGame"})
}
