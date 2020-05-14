package room

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/table"
)

// Client is a client connected to the server via websockets
type Client struct {
	// Conn is the underlying websocket connection
	Conn *websocket.Conn

	// Send is a channel for sending messages to the client
	Send chan interface{}

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
		Send:   make(chan interface{}, 256),
		Close:  make(chan string),
		Conn:   conn,
		player: player,
		table:  table,
	}
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
