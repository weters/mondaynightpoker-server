package room

import (
	"github.com/sirupsen/logrus"
)

// PitBoss is responsible for dispatching players to games
type PitBoss struct {
	dealers    map[string]*Dealer
	connect    chan *Client
	disconnect chan *Client
}

// NewPitBoss returns a new dispatch object
func NewPitBoss() *PitBoss {
	return &PitBoss{
		dealers:    make(map[string]*Dealer),
		connect:    make(chan *Client, 256),
		disconnect: make(chan *Client, 256),
	}
}

// StartShift starts the PitBoss run loop
func (p *PitBoss) StartShift() {
	go p.runLoop()
}

func (p *PitBoss) runLoop() {
	for {
		select {
		case client := <-p.connect:
			logrus.WithField("player", client.String()).Debug("client connected")
			dealer, found := p.dealers[client.table.UUID]
			if !found {
				dealer = NewDealer(p, client.table)
				dealer.StartShift()
				p.dealers[client.table.UUID] = dealer
			}

			dealer.AddClient(client)
		case client := <-p.disconnect:
			logrus.WithField("player", client.String()).Debug("client disconnected")
			dealer, found := p.dealers[client.table.UUID]
			if !found {
				logrus.WithField("uuid", client.table.UUID).WithField("type", "exception").Error("table not found")
				return
			}

			if dealer.RemoveClient(client) {
				dealer.EndShift()
				delete(p.dealers, client.table.UUID)
			}
		}
	}
}

// ClientConnected is called when a client connects to the server
func (p *PitBoss) ClientConnected(client *Client) {
	p.connect <- client
}

// ClientDisconnected is called when a client disconnects from the server
func (p *PitBoss) ClientDisconnected(client *Client) {
	p.disconnect <- client
}
