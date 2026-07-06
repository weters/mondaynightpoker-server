package room

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// disconnectCloseCode is the WebSocket close code sent when the server forces a
// client to disconnect (for example, when the client cannot keep up with the
// message stream). Clients must treat it as a signal to reconnect and resync.
const disconnectCloseCode = 4002

// Client is a client connected to the server via websockets
type Client struct {
	// Conn is the underlying websocket connection
	Conn *websocket.Conn

	// send is a channel for sending messages to the client
	send chan interface{}

	dealer         atomic.Pointer[Dealer]
	disconnectOnce sync.Once

	player *model.Player
	table  *model.Table
}

// NewClient returns a new client object
func NewClient(conn *websocket.Conn, player *model.Player, table *model.Table) *Client {
	return &Client{
		send:   make(chan interface{}, 256),
		Conn:   conn,
		player: player,
		table:  table,
	}
}

// Send sends a message to the web client
// If the client's send buffer is full, the client is disconnected so it can
// reconnect and resync rather than silently missing messages, and false is returned.
func (c *Client) Send(msg interface{}) bool {
	select {
	case c.send <- msg:
		return true
	default:
		c.Disconnect("send buffer full")
		return false
	}
}

// Disconnect sends the client a close frame with the provided reason and closes the
// underlying connection. It is safe to call from any goroutine; only the first call
// has an effect.
func (c *Client) Disconnect(reason string) {
	c.disconnectOnce.Do(func() {
		logrus.WithFields(logrus.Fields{
			"client": c.String(),
			"reason": reason,
		}).Warn("disconnecting client")

		if c.Conn == nil {
			return
		}

		msg := websocket.FormatCloseMessage(disconnectCloseCode, reason)
		_ = c.Conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
		_ = c.Conn.Close()
	})
}

// SendChan returns a read-only channel
func (c *Client) SendChan() <-chan interface{} {
	return c.send
}

// String returns a traceable identifier for the player and table
func (c *Client) String() string {
	email := "?"
	if c.player != nil {
		email = c.player.Email
	}

	tableUUID := "?"
	if c.table != nil {
		tableUUID = c.table.UUID
	}

	return fmt.Sprintf("%s:%s", email, tableUUID)
}

// setDealer atomically assigns the dealer responsible for this client
func (c *Client) setDealer(dealer *Dealer) {
	c.dealer.Store(dealer)
}

// ReceivedMessage is called when the server receives a message from a connected client
func (c *Client) ReceivedMessage(msg *playable.PayloadIn) {
	dealer := c.dealer.Load()
	if dealer == nil {
		logrus.WithField("msg", msg).Warn("received message, but dealer not found")
		return
	}

	dealer.ReceivedMessage(c, msg)
}
