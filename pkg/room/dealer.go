package room

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/room/gamefactory"
	"mondaynightpoker-server/pkg/table"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type state int

const (
	stateClientEvent state = iota
	stateGameEvent
	stateGameEnded
	stateGameScheduled
)

type action string

const (
	actionAdmin     action = "admin"
	actionStart     action = "start"
	actionRestart   action = "restart"
	actionTerminate action = "terminate"
)

// Dealer is responsible for controller the game
type Dealer struct {
	pitBoss *PitBoss
	table   *table.Table
	clients map[*Client]bool
	game    playable.Playable
	ticker  *time.Ticker

	execInRunLoop chan func()
	stateChanged  chan state
	close         chan bool

	// note: this must only be manipulated within the run loop
	logMessages []*playable.LogMessage

	pendingGame *pendingGame
}

// NewDealer creates a new dealer object
// This is called from a blocking state, so it needs to return quickly
func NewDealer(pitBoss *PitBoss, table *table.Table) *Dealer {
	d := &Dealer{
		pitBoss:       pitBoss,
		table:         table,
		clients:       make(map[*Client]bool),
		execInRunLoop: make(chan func(), 256),
		stateChanged:  make(chan state, 256),
		close:         make(chan bool),
		game:          nil,
	}

	return d
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
		var logChan <-chan []*playable.LogMessage
		if d.game != nil {
			logChan = d.game.LogChan()
		}

		var pendingGameTimer <-chan time.Time
		if d.pendingGame != nil {
			pendingGameTimer = d.pendingGame.timer.C
		}

		var ticker <-chan time.Time
		if d.ticker != nil {
			ticker = d.ticker.C
		}

		select {
		case <-ticker:
			if d.game != nil {
				if game, ok := d.game.(playable.Tickable); ok {
					if update, err := game.Tick(); err != nil {
						logrus.WithError(err).Error("Tick() failed")
					} else if update {
						d.sendGameData()
					}
				}

				if details, gameIsOver := d.game.GetEndOfGameDetails(); gameIsOver {
					if err := d.endGame(d.game, details); err != nil {
						logrus.WithError(err).Error("could end game")
					}
				}
			}
		case <-pendingGameTimer:
			if err := d.createGame(d.pendingGame.client, d.pendingGame.message); err != nil {
				d.pendingGame.client.Send(playable.Response{
					Key:   "error",
					Value: err.Error(),
				})
			}

			d.pendingGame = nil
		case messages := <-logChan:
			d.sendLogMessages(messages)
		case s := <-d.stateChanged:
			switch s {
			case stateClientEvent:
				d.sendPlayerData()
			case stateGameEvent:
				d.sendGameData()
			case stateGameEnded:
				d.sendGameEnded()
				d.sendPlayerData()
			case stateGameScheduled:
				d.sendGameScheduled()
			}
		case fn := <-d.execInRunLoop:
			fn()
		case <-d.close:
			log.WithField("uuid", d.table.UUID).Debug("terminating dealer run loop")
			return
		}
	}
}

// AddClient adds a client
// This method must return quickly
func (d *Dealer) AddClient(client *Client) {
	client.dealer = d
	d.clients[client] = true

	d.stateChanged <- stateClientEvent
	d.execInRunLoop <- func() {
		client.Send(playable.Response{
			Key:   "allLogs",
			Value: "",
			Data:  d.logMessages,
		})

		if d.pendingGame != nil {
			client.Send(playable.Response{
				Key:  "scheduledGame",
				Data: d.pendingGame,
			})
		}

		if d.game == nil {
			return
		}

		gs, err := d.game.GetPlayerState(client.player.ID)
		if err != nil {
			logrus.WithError(err).Error("could not get player state")
			return
		}

		client.Send(gs)
	}
}

// RemoveClient adds a client
// This method must return quickly
func (d *Dealer) RemoveClient(client *Client) (lastClient bool) {
	delete(d.clients, client)
	nClients := len(d.clients)

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
	for client := range d.clients {
		client.Send(playable.Response{
			Key: "gameEnded",
		})
	}
}

// NOTE: must only be called from the run loop
func (d *Dealer) sendGameData() {
	if d.game == nil {
		// should not happen
		logrus.Error("XXX game state changed, but there's no active game")
	}

	for client := range d.clients {
		data, err := d.game.GetPlayerState(client.player.ID)
		if err != nil {
			logrus.WithError(err).Error("could not get player state")
			continue
		}

		client.Send(data)
	}
}

func (d *Dealer) sendGameScheduled() {
	pendingGame := d.pendingGame
	for client := range d.clients {
		client.Send(playable.Response{
			Key:  "scheduledGame",
			Data: pendingGame,
		})
	}
}

func (d *Dealer) sendLogMessages(messages []*playable.LogMessage) {
	var gameName string
	if d.game != nil {
		gameName = d.game.Name()
	}

	for _, message := range messages {
		logrus.WithFields(logrus.Fields{
			"cards":     message.Cards,
			"playerIds": message.PlayerIDs,
			"tableId":   d.table.UUID,
			"game":      gameName,
			"message":   message.Message,
		}).Debug("log sent")
	}

	d.addLogMessages(messages)
	for client := range d.clients {
		client.Send(playable.Response{
			Key:   "logs",
			Value: "",
			Data:  messages,
		})
	}
}

func (d *Dealer) sendPlayerData() {
	players, err := d.table.GetPlayers(context.Background())
	if err != nil {
		logrus.WithField("uuid", d.table.UUID).WithError(err).Error("could not get players")
		return
	}

	connectedClients := make(map[int64]*table.Player)
	for client := range d.clients {
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

	for client := range d.clients {
		client.Send(playable.Response{
			Key:  "clientState",
			Data: csPlayers,
		})
	}
}

// canAdminOrSendError will send an error message to the client if they are not a table admin or site admin
// If they are an appropriate admin, true is returned, otherwise false is returned
func canPerformActionOnTable(ctx string, c *Client, action action) bool {
	if c.player.IsSiteAdmin {
		return true
	}

	playerTable, err := c.player.GetPlayerTable(context.Background(), c.table)
	if err != nil {
		c.Send(newErrorResponse(ctx, err))
		return false
	}

	if playerTable.IsTableAdmin {
		return true
	}

	switch action {
	case actionStart:
		if playerTable.CanStart {
			return true
		}
	case actionRestart:
		if playerTable.CanRestart {
			return true
		}
	case actionTerminate:
		if playerTable.CanTerminate {
			return true
		}
	case actionAdmin:
		// if you get here, you do not have permission
	default:
		logrus.WithField("action", action).Error("unknown action")
	}

	c.Send(newErrorResponse(ctx, errors.New("you do not have the appropriate permission")))
	return false
}

// ReceivedMessage is called when a client sends a message to the server
// IMPORTANT! this method MUST not access or modify any dealer state information
// Instead, it must run any operates within the run loop, using the execInRunLoop chan.
func (d *Dealer) ReceivedMessage(c *Client, msg *playable.PayloadIn) {
	if msgBytes, _ := json.Marshal(msg); msgBytes != nil {
		logrus.WithField("message", string(msgBytes)).Debug("client message")
	}

	switch msg.Action {
	case "cancelGame":
		if !canPerformActionOnTable(msg.Context, c, actionStart) {
			return
		}

		d.execInRunLoop <- func() {
			if d.pendingGame == nil {
				return
			}

			if !d.pendingGame.timer.Stop() {
				<-d.pendingGame.timer.C
			}

			d.pendingGame = nil
			d.stateChanged <- stateGameScheduled
			c.Send(playable.OK(msg.Context))
		}
	case "createGame":
		if d.game != nil {
			if !canPerformActionOnTable(msg.Context, c, actionRestart) {
				return
			}
		} else {
			if !canPerformActionOnTable(msg.Context, c, actionStart) {
				return
			}
		}

		d.execInRunLoop <- func() {
			if err := d.scheduleGame(c, msg); err != nil {
				c.Send(newErrorResponse(msg.Context, err))
				return
			}

			c.Send(playable.OK(msg.Context))
		}
	case "terminateGame":
		if !canPerformActionOnTable(msg.Context, c, actionTerminate) {
			return
		}

		d.execInRunLoop <- func() {
			d.unsetGame()
			d.stateChanged <- stateGameEnded
			d.sendLogMessages([]*playable.LogMessage{
				{
					UUID:      uuid.New().String(),
					PlayerIDs: []int64{c.player.ID},
					Cards:     nil,
					Message:   "{} ended the game early",
					Time:      time.Now(),
				},
			})
		}

		c.Send(playable.OK(msg.Context))
	case "tableAdmin":
		d.execInRunLoop <- func() {
			if !canPerformActionOnTable(msg.Context, c, actionAdmin) {
				return
			}

			playerID, ok := msg.AdditionalData["playerId"].(float64)
			if !ok {
				c.Send(newErrorResponse(msg.Context, errors.New("could not obtain playerId")))
				return
			}

			player, err := table.GetPlayerByID(context.Background(), int64(playerID))
			if err != nil {
				c.Send(newErrorResponse(msg.Context, err))
				return
			}

			playerTable, err := player.GetPlayerTable(context.Background(), c.table)
			if err != nil {
				c.Send(newErrorResponse(msg.Context, err))
				return
			}

			if isTableAdmin, ok := msg.AdditionalData["isTableAdmin"].(bool); ok {
				playerTable.IsTableAdmin = isTableAdmin
			}

			if canStart, ok := msg.AdditionalData["canStart"].(bool); ok {
				playerTable.CanStart = canStart
			}

			if canRestart, ok := msg.AdditionalData["canRestart"].(bool); ok {
				playerTable.CanRestart = canRestart
			}

			if canTerminate, ok := msg.AdditionalData["canTerminate"].(bool); ok {
				playerTable.CanTerminate = canTerminate
			}

			if isBlocked, ok := msg.AdditionalData["isBlocked"].(bool); ok {
				if isBlocked {
					playerTable.Active = false
				}

				playerTable.IsBlocked = isBlocked
			}

			if err := playerTable.Save(context.Background()); err != nil {
				c.Send(newErrorResponse(msg.Context, err))
				return
			}

			c.Send(playable.OK(msg.Context))
			d.stateChanged <- stateClientEvent
		}
	case "playerStatus":
		d.execInRunLoop <- func() {
			var pt *table.PlayerTable
			var err error

			// set status for other player, requires table admin
			playerID, ok := msg.AdditionalData["playerId"].(float64)
			if ok {
				if !canPerformActionOnTable(msg.Context, c, actionAdmin) {
					return
				}

				var player *table.Player
				player, err = table.GetPlayerByID(context.Background(), int64(playerID))
				if err != nil {
					c.Send(newErrorResponse(msg.Context, err))
					return
				}

				pt, err = player.GetPlayerTable(context.Background(), c.table)
			} else {
				// set status for self
				pt, err = c.player.GetPlayerTable(context.Background(), c.table)
			}

			if err != nil {
				c.Send(newErrorResponse(msg.Context, err))
				return
			}

			isActive, ok := msg.AdditionalData["active"].(bool)
			if !ok {
				c.Send(newErrorResponse(msg.Context, errors.New("active is not boolean")))
				return
			}

			if pt.IsBlocked && isActive {
				c.Send(newErrorResponse(msg.Context, errors.New("player is currently blocked from participating")))
				return
			}

			pt.Active = isActive

			if err := pt.Save(context.Background()); err != nil {
				c.Send(newErrorResponse(msg.Context, errors.New("active is not boolean")))
				return
			}

			c.Send(playable.OK(msg.Context))
			d.stateChanged <- stateClientEvent
		}
	default:
		if game := d.game; game != nil {
			action, updateState, err := game.Action(c.player.ID, msg)
			if err != nil {
				logrus.WithError(err).WithField("client", c.String()).Error("could not perform action")
				c.Send(newErrorResponse(msg.Context, err))
				return
			}

			if action != nil {
				action.Context = msg.Context
				c.Send(action)
			}

			if updateState {
				d.stateChanged <- stateGameEvent
			}

			if details, isOver := game.GetEndOfGameDetails(); isOver {
				if err := d.endGame(game, details); err != nil {
					c.Send(newErrorResponse(msg.Context, err))
					return
				}
			}

			return
		}

		logrus.WithField("msg", msg).Warn("unknown message")
	}
}

func (d *Dealer) endGame(game playable.Playable, details *playable.GameOverDetails) error {
	record, err := d.table.CreateGame(context.Background(), game.Name())
	if err != nil {
		return fmt.Errorf("could not create game: %w", err)
	}

	if err := record.EndGame(context.Background(), details.Log, details.BalanceAdjustments); err != nil {
		return fmt.Errorf("could not save game: %w", err)
	}

	d.unsetGame()
	d.stateChanged <- stateGameEnded
	return nil
}

func (d *Dealer) getNextPlayersIDsForGame() ([]int64, error) {
	players, err := d.table.GetActivePlayersShifted(context.Background())
	if err != nil {
		return nil, err
	}

	playerIDs := make([]int64, 0, len(players))
	for _, player := range players {
		if player.IsPlaying() {
			playerIDs = append(playerIDs, player.PlayerID)
		}
	}

	return playerIDs, nil
}

func (d *Dealer) scheduleGame(c *Client, msg *playable.PayloadIn) error {
	if d.pendingGame != nil {
		return errors.New("a game is already scheduled to start")
	}

	pendingGame, err := newPendingGame(c, msg)
	if err != nil {
		return err
	}

	d.pendingGame = pendingGame
	d.stateChanged <- stateGameScheduled
	return nil
}

func (d *Dealer) createGame(client *Client, msg *playable.PayloadIn) error {
	factory, err := gamefactory.Get(msg.Subject)
	if err != nil {
		return fmt.Errorf("game not found: %s", msg.Subject)
	}

	playerIDs, err := d.getNextPlayersIDsForGame()
	if err != nil {
		return err
	}

	details, _, err := factory.Details(msg.AdditionalData)
	if err != nil {
		return err
	}

	logger := logrus.WithFields(logrus.Fields{
		"startedBy": client.player.ID,
		"game":      details,
		"table":     d.table.UUID,
		"playerIDs": playerIDs,
	})

	game, err := factory.CreateGame(logger, playerIDs, msg.AdditionalData)
	if err != nil {
		return err
	}
	logger.Info("game started")

	d.game = game

	if t, ok := game.(playable.Tickable); ok {
		d.ticker = time.NewTicker(t.Delay())
	}

	d.stateChanged <- stateGameEvent
	return nil
}

func (d *Dealer) unsetGame() {
	if game := d.game; game != nil {
	LOG:
		for {
			select {
			case msgs := <-game.LogChan():
				d.sendLogMessages(msgs)
			default:
				break LOG
			}
		}
	}

	d.game = nil

	if d.ticker != nil {
		d.ticker.Stop()
		d.ticker = nil
	}
}
