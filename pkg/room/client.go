package room

import (
	"fmt"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/table"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Client is a client connected to the server via websockets
type Client struct {
	// Conn is the underlying websocket connection
	Conn *websocket.Conn

	// send is a channel for sending messages to the client
	send chan interface{}

	// Close is a channel for closing the client
	Close chan string

	// CloseError contains the reason why the connection was closed
	CloseError error

	dealer *Dealer

	player *table.Player
	table  *table.Table
}

// NewClient returns a new client object
func NewClient(conn *websocket.Conn, player *table.Player, table *table.Table) *Client {
	return &Client{
		send:   make(chan interface{}, 256),
		Close:  make(chan string),
		Conn:   conn,
		player: player,
		table:  table,
	}
}

// Send send a message to the web client
func (c *Client) Send(msg interface{}) bool {
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

// SendChan returns a read-only channel
func (c *Client) SendChan() <-chan interface{} {
	return c.send
}

// String returns a traceable identifier for the player and table
func (c *Client) String() string {
	return fmt.Sprintf("%s:%s", c.player.Email, c.table.UUID)
}

// ReceivedMessage is called when the server receives a message from a connected client
func (c *Client) ReceivedMessage(msg *playable.PayloadIn) {
	if c.dealer == nil {
		logrus.WithField("msg", msg).Warn("received message, but dealer not found")
		return
	}

	c.dealer.ReceivedMessage(c, msg)
}
