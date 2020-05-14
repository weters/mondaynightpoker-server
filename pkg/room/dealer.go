package room

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/bourre"
	"mondaynightpoker-server/pkg/table"
	"sync"
)

type state int

const (
	stateClientEvent state = iota
	stateGameEvent
	stateGameEnded
)

// Dealer is responsible for controller the game
type Dealer struct {
	pitBoss *PitBoss
	table   *table.Table
	clients map[*Client]bool
	lock    sync.RWMutex
	game    playable.Playable

	execInRunLoop            chan func()
	execInRunLoopWithClients chan func([]*Client)
	stateChanged             chan state
	close                    chan bool
}

// NewDealer creates a new dealer object
// This is called from a blocking state, so it needs to return quickly
func NewDealer(pitBoss *PitBoss, table *table.Table) *Dealer {
	d := &Dealer{
		pitBoss:                  pitBoss,
		table:                    table,
		clients:                  make(map[*Client]bool),
		execInRunLoop:            make(chan func(), 256),
		execInRunLoopWithClients: make(chan func([]*Client), 256),
		stateChanged:             make(chan state, 256),
		close:                    make(chan bool),
		game:                     nil,
	}

	return d
}

// Clients will return a slice of connected (at the time) clients
func (d *Dealer) Clients() []*Client {
	d.lock.RLock()
	defer d.lock.RUnlock()

	clients := make([]*Client, 0, len(d.clients))
	for client := range d.clients {
		clients = append(clients, client)
	}

	return clients
}

// StartShift starts the run loop
func (d *Dealer) StartShift() {
	go d.runLoop()
}

func (d *Dealer) runLoop() {
	log := logrus.WithFields(logrus.Fields{
		"uuid": d.table.UUID,
		"name": d.table.Name,
	})

	log.WithField("uuid", d.table.UUID).Debug("creating dealer run loop")
	for {
		select {
		case s := <-d.stateChanged:
			switch s {
			case stateClientEvent:
				d.sendPlayerData()
			case stateGameEvent:
				d.sendGameData()
			case stateGameEnded:
				d.sendGameEnded()
				d.sendPlayerData()
			}
		case fn := <-d.execInRunLoop:
			fn()
		case fn := <-d.execInRunLoopWithClients:
			d.lock.RLock()
			clients := make([]*Client, 0, len(d.clients))
			for client := range d.clients {
				clients = append(clients, client)
			}
			d.lock.RUnlock()

			fn(clients)
		case <-d.close:
			log.WithField("uuid", d.table.UUID).Debug("terminating dealer run loop")
			return
		}
	}
}

// AddClient adds a client
// This method must return quickly
func (d *Dealer) AddClient(client *Client) {
	d.lock.Lock()
	client.dealer = d
	d.clients[client] = true
	d.lock.Unlock()

	d.stateChanged <- stateClientEvent
	d.execInRunLoop <- func() {
		if d.game == nil {
			return
		}

		gs, err := d.game.GetPlayerState(client.player.ID)
		if err != nil {
			logrus.WithError(err).Error("could not get player state")
			return
		}

		client.Send <- gs
	}
}

// RemoveClient adds a client
// This method must return quickly
func (d *Dealer) RemoveClient(client *Client) (lastClient bool) {
	d.lock.Lock()
	delete(d.clients, client)
	nClients := len(d.clients)
	d.lock.Unlock()

	if nClients > 0 {
		d.stateChanged <- stateClientEvent
		return false
	}

	return true
}

// EndShift is called when the dealer is no longer needed
func (d *Dealer) EndShift() {
	close(d.close)
}

// NOTE: must only be called from the run loop
func (d *Dealer) sendGameEnded() {
	for _, client := range d.Clients() {
		client.Send <- &playable.Response{
			Key: "gameEnded",
		}
	}
}

// NOTE: must only be called from the run loop
func (d *Dealer) sendGameData() {
	if d.game == nil {
		// should not happen
		logrus.Error("XXX game state changed, but there's no active game")
	}

	for _, client := range d.Clients() {
		data, err := d.game.GetPlayerState(client.player.ID)
		if err != nil {
			logrus.WithError(err).Error("could not get player state")
			continue
		}

		client.Send <- data
	}
}

func (d *Dealer) sendPlayerData() {
	players, err := d.table.GetPlayers(context.Background())
	if err != nil {
		logrus.WithField("uuid", d.table.UUID).WithError(err).Error("could not get players")
		return
	}

	connectedClients := make(map[int64]*table.Player)
	for _, client := range d.Clients() {
		connectedClients[client.player.ID] = client.player
	}

	csPlayers := make(map[int64]*clientStatePlayers)
	for _, player := range players {
		_, isConnected := connectedClients[player.PlayerID]
		delete(connectedClients, player.PlayerID)
		csPlayers[player.PlayerID] = &clientStatePlayers{
			PlayerTable: player,
			IsConnected: isConnected,
			IsSeated:    true,
		}
	}

	for _, player := range connectedClients {
		csPlayers[player.ID] = &clientStatePlayers{
			PlayerTable: &table.PlayerTable{
				Player:    player,
				PlayerID:  player.ID,
				TableUUID: d.table.UUID,
			},
			IsConnected: true,
			IsSeated:    false,
		}
	}

	for _, client := range d.Clients() {
		client.Send <- &playable.Response{
			Key:  "clientState",
			Data: csPlayers,
		}
	}
}

// canAdminOrSendError will send an error message to the client if they are not a table admin or site admin
// If they are an appropriate admin, true is returned, otherwise false is returned
func canAdminTable(ctx string, c *Client) bool {
	if c.player.IsSiteAdmin {
		return true
	}

	playerTable, err := c.player.GetPlayerTable(context.Background(), c.table)
	if err != nil {
		c.Send <- newErrorResponse(ctx, err)
		return false
	}

	if !playerTable.IsTableAdmin {
		c.Send <- newErrorResponse(ctx, errors.New("you do not have the appropriate permission"))
		return false
	}

	return true
}

// ReceivedMessage is called when a client sends a message to the server
func (d *Dealer) ReceivedMessage(c *Client, msg *playable.PayloadIn) {
	switch msg.Action {
	case "createGame":
		if !canAdminTable(msg.Context, c) {
			return
		}

		switch msg.Subject {
		case "bourre":
			d.execInRunLoop <- func() {
				if err := d.createBourreGame(msg.AdditionalData); err != nil {
					c.Send <- newErrorResponse(msg.Context, err)
					return
				}

				c.Send <- playable.OK(msg.Context)

				return
			}
		default:
			// handle error
		}
	case "terminateGame":
		if !canAdminTable(msg.Context, c) {
			return
		}

		d.execInRunLoop <- func() {
			d.game = nil
			d.stateChanged <- stateGameEnded
		}

		c.Send <- playable.OK(msg.Context)
	case "tableAdmin":
		d.execInRunLoop <- func() {
			if !canAdminTable(msg.Context, c) {
				return
			}

			isTableAdmin, ok := msg.AdditionalData["isTableAdmin"].(bool)
			if !ok {
				c.Send <- newErrorResponse(msg.Context, errors.New("isTableAdmin is not boolean"))
				return
			}

			playerID, ok := msg.AdditionalData["playerId"].(float64)
			if !ok {
				c.Send <- newErrorResponse(msg.Context, errors.New("could not obtain playerId"))
				return
			}

			player, err := table.GetPlayerByID(context.Background(), int64(playerID))
			if err != nil {
				c.Send <- newErrorResponse(msg.Context, err)
				return
			}

			playerTable, err  := player.GetPlayerTable(context.Background(), c.table)
			if err != nil {
				c.Send <- newErrorResponse(msg.Context, err)
				return
			}

			if err := playerTable.SetIsTableAdmin(context.Background(), isTableAdmin); err != nil {
				c.Send <- newErrorResponse(msg.Context, err)
				return
			}

			c.Send <- playable.OK(msg.Context)
			d.stateChanged <- stateClientEvent
			return
		}
	case "playerStatus":
		d.execInRunLoop <- func() {
			var pt *table.PlayerTable
			var err error

			// set status for other player, requires table admin
			playerID, ok := msg.AdditionalData["playerId"].(float64)
			if ok {
				if !canAdminTable(msg.Context, c) {
					return
				}

				var player *table.Player
				player, err = table.GetPlayerByID(context.Background(), int64(playerID))
				if err != nil {
					c.Send <- newErrorResponse(msg.Context, err)
					return
				}

				pt, err = player.GetPlayerTable(context.Background(), c.table)
			} else {
				// set status for self
				pt, err = c.player.GetPlayerTable(context.Background(), c.table)
			}

			if err != nil {
				c.Send <- newErrorResponse(msg.Context, err)
				return
			}

			isActive, ok := msg.AdditionalData["active"].(bool)
			if !ok {
				c.Send <- newErrorResponse(msg.Context, errors.New("active is not boolean"))
				return
			}

			if err := pt.SetActive(context.Background(), isActive); err != nil {
				c.Send <- newErrorResponse(msg.Context, errors.New("active is not boolean"))
				return
			}

			c.Send <- playable.OK(msg.Context)
			d.stateChanged <- stateClientEvent
		}
	default:
		if game := d.game; game != nil {
			action, updateState, err := game.Action(c.player.ID, msg)
			if err != nil {
				logrus.WithError(err).WithField("client", c.String()).Error("could not perform action")
				c.Send <- newErrorResponse(msg.Context, err)
				return
			}

			if action != nil {
				action.Context = msg.Context
				c.Send <- action
			}

			if updateState {
				d.stateChanged <- stateGameEvent
			}

			if details, isOver := game.GetEndOfGameDetails(); isOver {
				record, err := d.table.CreateGame(context.Background(), game.Name())
				if err != nil {
					logrus.WithError(err).Error("could not create game")
					c.Send <- newErrorResponse(msg.Context, err)
					return
				}

				if err := record.EndGame(context.Background(), details.Log, details.BalanceAdjustments); err != nil {
					logrus.WithError(err).Error("could not save game")
					c.Send <- newErrorResponse(msg.Context, err)
					return
				}

				d.game = nil
				d.stateChanged <- stateGameEnded
			}

			return
		}

		logrus.WithField("msg", msg).Warn("unknown message")
	}
}

func (d *Dealer) createBourreGame(additionalData map[string]interface{}) error {
	players, err := d.table.GetPlayersShifted(context.Background())
	if err != nil {
		return err
	}

	playerIDs := make([]int64, 0, len(players))
	for _, player := range players {
		if player.Active {
			playerIDs = append(playerIDs, player.PlayerID)
		}
	}

	ante := 0
	if rawAnte, ok := additionalData["ante"]; ok {
		ante = int(rawAnte.(float64))
	}

	game, err := bourre.NewGame(d.table.UUID, playerIDs, bourre.Options{Ante: ante})
	if err != nil {
		return err
	}

	if err := game.Deal(); err != nil {
		return err
	}

	d.game = game
	d.stateChanged <- stateGameEvent

	return nil
}
