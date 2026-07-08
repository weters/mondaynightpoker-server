package room

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/room/gamefactory"
	"sort"
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
	pitBoss        *PitBoss
	store          TableStore
	startGameDelay time.Duration
	table          *model.Table
	// note: clients must only be accessed within the run loop
	clients map[*Client]bool
	game    playable.Playable
	ticker  *time.Ticker

	execInRunLoop chan func()
	stateChanged  chan state
	close         chan bool

	// persistRetryDelays are the waits between persistGameResult attempts
	persistRetryDelays []time.Duration

	// note: everything below must only be manipulated within the run loop
	logMessages []*playable.LogMessage

	// rosterGen tags roster fetches so stale results are dropped
	rosterGen uint64
	// persistDone is closed once the previous game's results are saved; the
	// next game start waits on it so balances are read post-adjustment
	persistDone chan struct{}

	pendingGame       *pendingGame
	lastGameStartedBy *model.Player
}

// NewDealer creates a new dealer object
// This is called from a blocking state, so it needs to return quickly
func NewDealer(pitBoss *PitBoss, table *model.Table) *Dealer {
	d := &Dealer{
		pitBoss:            pitBoss,
		store:              pitBoss.store,
		startGameDelay:     pitBoss.startGameDelay,
		table:              table,
		clients:            make(map[*Client]bool),
		execInRunLoop:      make(chan func(), 256),
		stateChanged:       make(chan state, 256),
		close:              make(chan bool),
		persistRetryDelays: []time.Duration{time.Second, 5 * time.Second},
		game:               nil,
	}

	return d
}

// StartShift starts the run loop
func (d *Dealer) StartShift() {
	go d.runLoop()
}

// tryExecInRunLoop re-enters the run loop from a background goroutine without
// ever blocking; if the dealer has been retired or the queue is full, the
// callback is dropped with a log line.
func (d *Dealer) tryExecInRunLoop(fn func()) {
	select {
	case d.execInRunLoop <- fn:
	default:
		logrus.WithField("uuid", d.table.UUID).Warn("dealer run loop unavailable; dropping callback")
	}
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
					d.endGame(d.game, details)
				}
			}
		case <-pendingGameTimer:
			pg := d.pendingGame
			d.pendingGame = nil
			d.startGame(pg.client, pg.message)
		case messages := <-logChan:
			d.sendLogMessages(messages)
		case s := <-d.stateChanged:
			switch s {
			case stateClientEvent:
				d.requestPlayerData()
			case stateGameEvent:
				d.sendGameData()
			case stateGameEnded:
				d.sendGameEnded()
				d.requestPlayerData()
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
	client.setDealer(d)

	d.execInRunLoop <- func() {
		d.clients[client] = true
		d.requestPlayerData()

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

		if rp, ok := d.game.(playable.RulesProvider); ok {
			gs.Rules = rp.Rules()
		}

		client.Send(gs)
	}
}

// RemoveClient removes a client and reports whether it was the dealer's last client
// This method blocks until the run loop processes the removal
func (d *Dealer) RemoveClient(client *Client) (lastClient bool) {
	res := make(chan bool, 1)
	d.execInRunLoop <- func() {
		delete(d.clients, client)
		if len(d.clients) > 0 {
			d.requestPlayerData()
			res <- false
			return
		}

		res <- true
	}

	return <-res
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

	var rules []playable.RuleSection
	if rp, ok := d.game.(playable.RulesProvider); ok {
		rules = rp.Rules()
	}

	for client := range d.clients {
		data, err := d.game.GetPlayerState(client.player.ID)
		if err != nil {
			logrus.WithError(err).Error("could not get player state")
			continue
		}

		data.Rules = rules
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

// requestPlayerData fetches the roster on a background goroutine and re-enters
// the run loop to send it. Stale fetches (an older one finishing after a newer
// one started) are dropped via the generation counter.
// NOTE: must only be called from the run loop
func (d *Dealer) requestPlayerData() {
	d.rosterGen++
	gen := d.rosterGen

	go func() {
		players, err := d.store.GetPlayers(context.Background(), d.table)
		if err != nil {
			logrus.WithField("uuid", d.table.UUID).WithError(err).Error("could not get players")
			return
		}

		d.tryExecInRunLoop(func() {
			if gen != d.rosterGen {
				return // a newer roster fetch superseded this one
			}

			d.sendPlayerData(players)
		})
	}()
}

// sendPlayerData sends the client state built from a fetched roster
// NOTE: must only be called from the run loop
func (d *Dealer) sendPlayerData(players []*model.PlayerTable) {
	connectedClients := make(map[int64]*model.Player)
	for client := range d.clients {
		connectedClients[client.player.ID] = client.player
	}

	var nextPickerID int64
	if d.lastGameStartedBy != nil {
		if next := getNextPicker(players, d.lastGameStartedBy.ID); next != nil {
			nextPickerID = next.PlayerID
		}
	}

	csPlayers := make(map[int64]*clientStatePlayers)
	for _, player := range players {
		_, isConnected := connectedClients[player.PlayerID]
		delete(connectedClients, player.PlayerID)
		csPlayers[player.PlayerID] = &clientStatePlayers{
			PlayerTable:  player,
			IsConnected:  isConnected,
			IsSeated:     true,
			IsNextPicker: nextPickerID != 0 && player.PlayerID == nextPickerID,
		}
	}

	for _, player := range connectedClients {
		csPlayers[player.ID] = &clientStatePlayers{
			PlayerTable: &model.PlayerTable{
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

// fetchPlayerTable loads the caller's player-table record for permission
// checks. It runs on the caller's goroutine (the WebSocket read loop) so the
// run loop never blocks on the database; site admins skip the lookup.
func (d *Dealer) fetchPlayerTable(c *Client) (*model.PlayerTable, error) {
	if c.player.IsSiteAdmin {
		return nil, nil
	}

	return d.store.GetPlayerTable(context.Background(), c.player, c.table)
}

// canPerformAction is the pure permission check over a prefetched player-table
// record. It sends an error response to the client when permission is denied.
func canPerformAction(ctx string, c *Client, playerTable *model.PlayerTable, fetchErr error, action action) bool {
	if c.player.IsSiteAdmin {
		return true
	}

	if fetchErr != nil {
		c.Send(newErrorResponse(ctx, fetchErr))
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
		pt, fetchErr := d.fetchPlayerTable(c)
		if !canPerformAction(msg.Context, c, pt, fetchErr, actionStart) {
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
		pt, fetchErr := d.fetchPlayerTable(c)

		d.execInRunLoop <- func() {
			// restarting over a running game needs a different permission, and
			// d.game may only be read inside the run loop
			required := actionStart
			if d.game != nil {
				required = actionRestart
			}

			if !canPerformAction(msg.Context, c, pt, fetchErr, required) {
				return
			}

			if err := d.scheduleGame(c, msg); err != nil {
				c.Send(newErrorResponse(msg.Context, err))
				return
			}

			c.Send(playable.OK(msg.Context))
		}
	case "terminateGame":
		pt, fetchErr := d.fetchPlayerTable(c)
		if !canPerformAction(msg.Context, c, pt, fetchErr, actionTerminate) {
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
		ownPT, fetchErr := d.fetchPlayerTable(c)
		if !canPerformAction(msg.Context, c, ownPT, fetchErr, actionAdmin) {
			return
		}

		playerID, ok := msg.AdditionalData["playerId"].(float64)
		if !ok {
			c.Send(newErrorResponse(msg.Context, errors.New("could not obtain playerId")))
			return
		}

		player, err := d.store.GetPlayerByID(context.Background(), int64(playerID))
		if err != nil {
			c.Send(newErrorResponse(msg.Context, err))
			return
		}

		playerTable, err := d.store.GetPlayerTable(context.Background(), player, c.table)
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

		if err := d.store.SavePlayerTable(context.Background(), playerTable); err != nil {
			c.Send(newErrorResponse(msg.Context, err))
			return
		}

		c.Send(playable.OK(msg.Context))
		d.stateChanged <- stateClientEvent
	case "tableStake":
		pt, err := d.store.GetPlayerTable(context.Background(), c.player, c.table)
		if err != nil {
			c.Send(newErrorResponse(msg.Context, err))
			return
		}

		tableStake, ok := msg.AdditionalData["tableStake"].(float64)
		if !ok {
			c.Send(newErrorResponse(msg.Context, errors.New("tableStake not passed in")))
			return
		}

		const minTableStake = 500
		const maxTableStake = 10_000

		if tableStake < minTableStake || tableStake > maxTableStake {
			c.Send(newErrorResponse(msg.Context, fmt.Errorf("tableStake must be >= ${%d} and <= ${%d}", minTableStake, maxTableStake)))
			return
		}

		pt.TableStake = int(tableStake)
		if err := d.store.SavePlayerTable(context.Background(), pt); err != nil {
			c.Send(newErrorResponse(msg.Context, err))
			return
		}

		c.Send(playable.OK(msg.Context))
		d.stateChanged <- stateClientEvent
	case "playerStatus":
		var pt *model.PlayerTable
		var err error

		// set status for other player, requires table admin
		playerID, ok := msg.AdditionalData["playerId"].(float64)
		if ok {
			ownPT, fetchErr := d.fetchPlayerTable(c)
			if !canPerformAction(msg.Context, c, ownPT, fetchErr, actionAdmin) {
				return
			}

			var player *model.Player
			player, err = d.store.GetPlayerByID(context.Background(), int64(playerID))
			if err != nil {
				c.Send(newErrorResponse(msg.Context, err))
				return
			}

			pt, err = d.store.GetPlayerTable(context.Background(), player, c.table)
		} else {
			// set status for self
			pt, err = d.store.GetPlayerTable(context.Background(), c.player, c.table)
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
		if err := d.store.SavePlayerTable(context.Background(), pt); err != nil {
			c.Send(newErrorResponse(msg.Context, err))
			return
		}

		c.Send(playable.OK(msg.Context))
		d.stateChanged <- stateClientEvent
	default:
		d.execInRunLoop <- func() {
			game := d.game
			if game == nil {
				logrus.WithField("msg", msg).Warn("unknown message")
				return
			}

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
				d.sendGameData()
			}

			if details, isOver := game.GetEndOfGameDetails(); isOver {
				d.endGame(game, details)
			}
		}
	}
}

// endGame tears the game down immediately and persists the results on a
// background goroutine. startGame waits on persistDone before reading the
// roster, so the next game always sees post-adjustment balances.
// NOTE: must only be called from the run loop
func (d *Dealer) endGame(game playable.Playable, details *playable.GameOverDetails) {
	gameName := game.Name()

	var lastPickerID int64
	if d.lastGameStartedBy != nil {
		lastPickerID = d.lastGameStartedBy.ID
	}

	d.unsetGame()
	d.stateChanged <- stateGameEnded

	done := make(chan struct{})
	d.persistDone = done

	go func() {
		persistErr := d.persistGameResult(gameName, details)
		if persistErr != nil {
			logrus.WithField("uuid", d.table.UUID).WithError(persistErr).Error("could not persist game result")
		}

		// release any waiting game start before the cosmetic picker fetch
		close(done)

		var pickerMsgs []*playable.LogMessage
		if lastPickerID > 0 {
			if players, err := d.store.GetPlayers(context.Background(), d.table); err == nil {
				pickerMsgs = buildPickerLogMessage(players, lastPickerID)
			}
		}

		d.tryExecInRunLoop(func() {
			if persistErr != nil {
				d.sendLogMessages([]*playable.LogMessage{{
					UUID:    uuid.New().String(),
					Message: "the game results could not be saved; balances may be out of date",
					Time:    time.Now(),
				}})
			}

			if len(pickerMsgs) > 0 {
				d.sendLogMessages(pickerMsgs)
			}

			d.requestPlayerData()
		})
	}()
}

// persistGameResult saves the game record and balance adjustments, retrying on
// failure. Runs on a background goroutine.
func (d *Dealer) persistGameResult(gameName string, details *playable.GameOverDetails) error {
	var lastErr error
	for attempt := 0; attempt <= len(d.persistRetryDelays); attempt++ {
		if attempt > 0 {
			time.Sleep(d.persistRetryDelays[attempt-1])
		}

		record, err := d.store.CreateGame(context.Background(), d.table, gameName)
		if err != nil {
			lastErr = fmt.Errorf("could not create game: %w", err)
			continue
		}

		if err := d.store.EndGame(context.Background(), record, details.Log, details.BalanceAdjustments); err != nil {
			lastErr = fmt.Errorf("could not save game: %w", err)
			continue
		}

		return nil
	}

	return lastErr
}

func buildPickerLogMessage(players []*model.PlayerTable, pickerID int64) []*playable.LogMessage {
	last, next := getLastAndNextPicker(players, pickerID)
	if last == nil || next == nil {
		return nil
	}

	return []*playable.LogMessage{
		{
			UUID:    uuid.New().String(),
			Message: fmt.Sprintf("%s picked the last game. %s is next to pick", last.Player.DisplayName, next.Player.DisplayName),
			Time:    time.Now(),
		},
	}
}

// getNextPicker returns the player who should pick the next game given the player who last picked.
// Returns nil if the last picker cannot be located among active players.
func getNextPicker(players []*model.PlayerTable, lastPickerID int64) *model.PlayerTable {
	_, next := getLastAndNextPicker(players, lastPickerID)
	return next
}

func getLastAndNextPicker(players []*model.PlayerTable, lastPickerID int64) (*model.PlayerTable, *model.PlayerTable) {
	activePlayers := make([]*model.PlayerTable, 0, len(players))
	for _, p := range players {
		if p.IsPlaying() {
			activePlayers = append(activePlayers, p)
		}
	}

	if len(activePlayers) == 0 {
		return nil, nil
	}

	sort.Slice(activePlayers, func(i, j int) bool {
		return activePlayers[i].Player.DisplayName < activePlayers[j].Player.DisplayName
	})

	pickerIndex := -1
	for i, p := range activePlayers {
		if p.PlayerID == lastPickerID {
			pickerIndex = i
			break
		}
	}

	if pickerIndex < 0 {
		return nil, nil
	}

	nextIndex := (pickerIndex + 1) % len(activePlayers)
	return activePlayers[pickerIndex], activePlayers[nextIndex]
}

func (d *Dealer) getNextPlayersForGame() ([]*model.PlayerTable, error) {
	players, err := d.store.GetActivePlayersShifted(context.Background(), d.table)
	if err != nil {
		return nil, err
	}

	filteredPlayers := make([]*model.PlayerTable, 0, len(players))
	for _, player := range players {
		if player.IsPlaying() {
			filteredPlayers = append(filteredPlayers, player)
		}
	}

	return filteredPlayers, nil
}

func (d *Dealer) scheduleGame(c *Client, msg *playable.PayloadIn) error {
	if d.pendingGame != nil {
		return errors.New("a game is already scheduled to start")
	}

	pendingGame, err := newPendingGame(c, msg, d.startGameDelay)
	if err != nil {
		return err
	}

	d.pendingGame = pendingGame
	d.stateChanged <- stateGameScheduled
	return nil
}

// startGame fetches the seating order in the background, then constructs the
// game back on the run loop. If the previous game's results are still being
// saved, the roster read waits for them so it sees post-adjustment balances.
// NOTE: must only be called from the run loop
func (d *Dealer) startGame(client *Client, msg *playable.PayloadIn) {
	persistDone := d.persistDone

	go func() {
		if persistDone != nil {
			<-persistDone
		}

		players, err := d.getNextPlayersForGame()

		d.tryExecInRunLoop(func() {
			if d.game != nil {
				return // a game started while the roster was being fetched
			}

			if err == nil {
				err = d.createGame(client, msg, players)
			}

			if err != nil {
				client.Send(playable.Response{
					Key:   "error",
					Value: err.Error(),
				})
			}
		})
	}()
}

// createGame constructs the game from a prefetched roster
// NOTE: must only be called from the run loop
func (d *Dealer) createGame(client *Client, msg *playable.PayloadIn, players []*model.PlayerTable) error {
	factory, err := gamefactory.Get(msg.Subject)
	if err != nil {
		return fmt.Errorf("game not found: %s", msg.Subject)
	}

	details, _, err := factory.Details(msg.AdditionalData)
	if err != nil {
		return err
	}

	playerIDs := make([]int64, len(players))
	for i, player := range players {
		playerIDs[i] = player.PlayerID
	}

	logger := logrus.WithFields(logrus.Fields{
		"startedBy": client.player.ID,
		"game":      details,
		"table":     d.table.UUID,
		"playerIDs": playerIDs,
	})

	game, err := factory.CreateGame(logger, players, msg.AdditionalData)
	if err != nil {
		return err
	}
	logger.Info("game started")

	d.game = game
	d.lastGameStartedBy = client.player

	if t, ok := game.(playable.Tickable); ok {
		d.ticker = time.NewTicker(t.Interval())
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
