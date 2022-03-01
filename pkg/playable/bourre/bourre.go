package bourre

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type playedCard struct {
	card   *deck.Card
	player *Player
}

// Game is a game of bourré
type Game struct {
	options       Options
	pot           int
	ante          int
	deck          *deck.Deck
	trumpCard     *deck.Card
	playerOrder   map[*Player]int
	idToPlayer    map[int64]*Player
	foldedPlayers map[*Player]bool

	parentResult *Result
	result       *Result // only populated when EndGame() is called

	// keep track of how many cards the player wants to trade. -1 means the player folds
	playerDiscards map[*Player][]*deck.Card

	// round data
	roundNo               int // round 0 is the trade-in round, rounds 1-5 are playing rounds, round 6 means game is over
	cardsPlayed           []*playedCard
	winningCardPlayed     *playedCard
	roundWinnerCalculated bool

	logger  logrus.FieldLogger
	logChan chan []*playable.LogMessage

	pendingDealerAction *pendingDealerAction

	// sendUpdate will send update if true
	sendUpdate bool

	// done will be set after the game is over and the user's have stated they want to proceed
	done bool
}

// Interval determines how often Tick() should be called
func (g *Game) Interval() time.Duration {
	return time.Second
}

// Tick will check the state of the game and possibly move the state along
func (g *Game) Tick() (bool, error) {
	if g.sendUpdate {
		g.sendUpdate = false
		return true, nil
	}

	if g.done {
		return false, nil
	}

	if g.pendingDealerAction != nil {
		if time.Now().After(g.pendingDealerAction.ExecuteAfter) {
			action := g.pendingDealerAction.Action
			switch action {
			case dealerActionReplaceDiscards:
				if err := g.replaceDiscards(); err != nil {
					logrus.WithError(err).Error("could not go to the next round")
				}
			case dealerActionNextRound:
				if err := g.nextRound(); err != nil {
					logrus.WithError(err).Error("could not go to the next round")
				}
			case dealerActionClearGame:
				g.done = true
			default:
				panic(fmt.Sprintf("unknown dealer action: %d", g.pendingDealerAction.Action))
			}

			g.pendingDealerAction = nil
			return true, nil
		}

		return false, nil
	}

	if g.isGameOver() {
		if err := g.endGame(); err != nil {
			logrus.WithError(err).Error("could not end game")
		}
	} else if g.isRoundOver() {
		action := dealerActionNextRound
		if g.roundNo == 0 {
			action = dealerActionReplaceDiscards
		}

		g.pendingDealerAction = &pendingDealerAction{
			Action:       action,
			ExecuteAfter: time.Now().Add(time.Second * 1),
		}
	}

	return false, nil
}

// Name returns "bourre"
func (g *Game) Name() string {
	return "bourre"
}

// LogChan returns a channel for sending log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}

// Action performs an action
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	player, ok := g.idToPlayer[playerID]
	if !ok {
		return nil, false, errors.New("player not found with that ID")
	}

	log := g.logger.WithField("playerID", playerID)

	switch message.Action {
	case "discard":
		log.WithField("cards", message.Cards).Debug("player discards")
		if err := g.playerDidDiscard(player, message.Cards); err != nil {
			return nil, false, err
		}

		if message.Cards == nil {
			g.sendLogMessages(newLogMessage(player.PlayerID, nil, "{} folded"))
		} else {
			g.sendLogMessages(newLogMessage(player.PlayerID, nil, "{} discarded %d", len(message.Cards)))
		}

		return playable.OK(), true, nil
	case "playCard":
		if len(message.Cards) != 1 {
			return nil, false, fmt.Errorf("expected to get 1 card, got %d", len(message.Cards))
		}

		log.WithField("card", message.Cards[0]).Debug("play card")
		if err := g.playerDidPlayCard(player, message.Cards[0]); err != nil {
			return nil, false, err
		}

		g.sendLogMessages(newLogMessage(player.PlayerID, message.Cards[0], "{} played a card"))
		return playable.OK(), true, nil
	default:
		return nil, false, fmt.Errorf("unknown action: %s", message.Action)
	}
}

func (g *Game) endGame() error {
	log := logrus.StandardLogger()

	res := g.result
	if res == nil {
		return errors.New("done cannot be called yet")
	}

	messages := make([]*playable.LogMessage, 0)

	if res.WinningAmount > 0 {
		messages = append(messages, newLogMessage(res.Winners[0].PlayerID, nil, "{} won ${%d}", res.WinningAmount))
	} else {
		messages = append(messages, newLogMessageWithPlayers(res.Winners, "{} tied for most tricks"))
	}

	if len(res.PaidPot) > 0 {
		messages = append(messages, newLogMessageWithPlayers(res.PaidPot, "{} pays the pot of ${%d}", res.OldPot))
	}

	if len(res.PaidAnte) > 0 {
		messages = append(messages, newLogMessageWithPlayers(res.PaidAnte, "{} pays the ante of ${%d}", res.Ante))
	}

	if len(res.Booted) > 0 {
		messages = append(messages, newLogMessageWithPlayers(res.Booted, "{} was booted"))
	}

	log.Debug("done triggered")
	if res.ShouldContinue() {
		log.Debug("new game created")
		game, err := res.NewGame()
		if err != nil {
			return err
		}

		if err := game.Deal(); err != nil {
			return err
		}

		messages = append(messages, newLogMessage(0, nil, "Another game is required"))

		*g = *game
		g.sendUpdate = true
	} else {
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionClearGame,
			ExecuteAfter: time.Now().Add(time.Second),
		}

		messages = append(messages, newLogMessage(0, nil, "The game ends"))
	}

	g.sendLogMessages(messages...)
	return nil
}

// GetEndOfGameDetails returns details at the end of the game
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	if !g.done {
		return nil, false
	}

	result := g.result
	if result == nil {
		return nil, false
	}

	if result.ShouldContinue() {
		return nil, false
	}

	adjustments := make(map[int64]int)
	for player := range g.playerOrder {
		adjustments[player.PlayerID] = player.balance
	}

	for player := range g.foldedPlayers {
		if _, found := adjustments[player.PlayerID]; found {
			panic("player should not also appear in folded")
		}

		adjustments[player.PlayerID] = player.balance
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: adjustments,
		Log:                result,
	}, true
}

// NewGame returns a new bourré game
// players should be in the correct order. i.e., any rotation must happen beforehand
func NewGame(logger logrus.FieldLogger, playerIDs []int64, opts Options) (*Game, error) {
	idToPlayer := make(map[int64]*Player)
	players := make([]*Player, len(playerIDs))
	for i, pid := range playerIDs {
		players[i] = NewPlayer(pid)
		idToPlayer[pid] = players[i]
	}

	g, err := newGame(logger, players, nil, opts)
	if err != nil {
		return nil, err
	}

	g.idToPlayer = idToPlayer
	return g, nil
}

func newGame(logger logrus.FieldLogger, players []*Player, foldedPlayers []*Player, opts Options) (*Game, error) {
	limit := 8
	if opts.FiveSuit {
		limit = 10
	}

	if len(players) < 2 || len(players) > limit {
		return nil, PlayerCountError{
			Max: limit,
			Got: len(players),
		}
	}

	pot := opts.InitialPot

	messages := make([]*playable.LogMessage, 0)

	playerOrder := make(map[*Player]int)
	for order, player := range players {
		// if initial pot is > 0, that means we are working off of a previous game. In that case,
		// we already took care of the players who need to ante
		if opts.InitialPot == 0 {
			messages = append(messages, newLogMessage(player.PlayerID, nil, "{} paid the ${%d} ante", opts.Ante))
			pot += opts.Ante
			player.balance -= opts.Ante
		}

		playerOrder[player] = order
	}

	var d *deck.Deck
	if opts.FiveSuit {
		d = deck.NewFiveSuit()
	} else {
		d = deck.New()
	}

	d.Shuffle()

	foldedPlayersMap := make(map[*Player]bool)
	for _, player := range foldedPlayers {
		foldedPlayersMap[player] = true
	}

	g := &Game{
		options:        opts,
		deck:           d,
		pot:            pot,
		ante:           opts.Ante,
		playerOrder:    playerOrder,
		foldedPlayers:  foldedPlayersMap,
		playerDiscards: make(map[*Player][]*deck.Card),
		logChan:        make(chan []*playable.LogMessage, 256),
		logger:         logger,
	}

	messages = append(messages, newLogMessage(0, nil, "New game of Bourré started with a pot of ${%d}", pot))
	g.sendLogMessages(messages...)

	return g, nil
}

// Deal will deal five cards to each player and select the bourré card
func (g *Game) Deal() error {
	if len(g.playerOrder) < 2 {
		return ErrNotEnoughActivePlayers
	}

	for i := 0; i < 5; i++ {
		for player := range g.playerOrder {
			card, err := g.deck.Draw()
			if err != nil {
				return err
			}

			player.AddCard(card)
		}
	}

	trumpCard, err := g.deck.Draw()
	if err != nil {
		return err
	}

	g.logger.WithField("trumpCard", trumpCard).Debug("trump card")

	g.sendLogMessages(newLogMessage(0, trumpCard, "The trump card has been selected"))

	g.trumpCard = trumpCard
	return nil
}

func (g *Game) maxDraw(player *Player) int {
	pos := g.playerOrder[player]
	if len(g.playerOrder) > 5 && pos < 5 {
		return 3
	}

	return 4
}

// playerDidDiscard determines which cards to discard
// If discards is nil, the player elects to fold
func (g *Game) playerDidDiscard(player *Player, discards []*deck.Card) error {
	if !g.isTradeInRound() {
		return ErrTradeInRoundIsOver
	}

	if g.isRoundOver() {
		return ErrRoundIsOver
	}

	if g.getCurrentTurn() != player {
		return ErrIsNotPlayersTurn
	}

	if discards == nil {
		// check if we are the last player to fold
		if len(g.playerDiscards)+1 == len(g.playerOrder) {
			hasPlayer := false
			for _, discards := range g.playerDiscards {
				if discards != nil {
					hasPlayer = true
					break
				}
			}

			if !hasPlayer {
				return ErrLastPlayerMustPlay
			}
		}

		g.playerDiscards[player] = nil
		player.Fold()
		return nil
	}

	maxDraw := g.maxDraw(player)
	if len(discards) > maxDraw {
		return fmt.Errorf("you cannot draw %d cards, the max for you is %d", len(discards), maxDraw)
	}

	duplicate := make(map[*deck.Card]bool)
	for _, card := range discards {
		if !player.HasCard(card) {
			return ErrCardNotInPlayersHand
		}

		if _, found := duplicate[card]; found {
			return ErrCannotDiscardTheSameCard
		}

		duplicate[card] = true
	}

	g.playerDiscards[player] = append([]*deck.Card{}, discards...)
	return nil
}

// replaceDiscards will replace the discarded cards with fresh ones
func (g *Game) replaceDiscards() error {
	if len(g.playerDiscards) != len(g.playerOrder) {
		return ErrRoundNotOver
	}

	discardPile := []*deck.Card{g.trumpCard}
	players := make([]*Player, len(g.playerOrder))
	for player, i := range g.playerOrder {
		players[i] = player

		if player.folded {
			discardPile = append(discardPile, player.hand...)
			player.hand = nil
		}
	}

	newPlayerOrder := make(map[*Player]int)
	i := 0
	for _, player := range players {
		if player.folded {
			g.foldedPlayers[player] = true
			continue
		}

		newPlayerOrder[player] = i
		i++

		discards, ok := g.playerDiscards[player]
		if !ok {
			panic(fmt.Sprintf("could not find player discards: %v", player))
		}

		for _, card := range discards {
			if err := player.playerDidPlayCard(card); err != nil {
				panic(err)
			}

			if !g.deck.CanDraw(1) {
				g.deck.ShuffleDiscards(discardPile)
			}

			newCard, err := g.deck.Draw()
			if err != nil {
				panic(err)
			}
			player.AddCard(newCard)
		}

		discardPile = append(discardPile, discards...)
	}

	g.playerOrder = newPlayerOrder
	g.roundNo++

	// only one left, end the game!
	if len(g.playerOrder) == 1 {
		return g.buildResults()
	}

	return nil
}

// canGameEnd determines whether the game can end
func (g *Game) canGameEnd() bool {
	return g.roundNo == 6 || len(g.playerOrder) == 1
}

// buildResults calculates the final results and stores it in the "results" attribute
func (g *Game) buildResults() error {
	if g.result != nil {
		return nil
	}

	if !g.canGameEnd() {
		return ErrGameNotOver
	}

	payPot := make([]*Player, 0)
	payAnte := make([]*Player, 0)
	winners := make([]*Player, 0)
	newPot := 0
	winningAmount := 0
	booted := make([]*Player, 0)

	if len(g.playerOrder) == 1 {
		for player := range g.playerOrder {
			player.balance += g.pot
			winners = append(winners, player)
		}

		winningAmount = g.pot
	} else {
		maxWins := 0

		for player := range g.playerOrder {
			if player.winCount == 0 {
				payPot = append(payPot, player)
			} else if player.winCount == maxWins {
				winners = append(winners, player)
			} else if player.winCount > maxWins {
				payAnte = append(payAnte, winners...)
				winners = []*Player{player}
				maxWins = player.winCount
			} else {
				payAnte = append(payAnte, player)
			}
		}

		// game continues
		if len(winners) > 1 || len(payPot) > 0 {
			// special 2-2-1 case. The "1" gets booted from the game
			if len(winners) == 2 && len(payAnte) == 1 && len(payPot) == 0 {
				booted = append(booted, payAnte[0])
				payAnte = []*Player{}
			}

			for _, player := range payPot {
				player.balance -= g.pot
				newPot += g.pot
			}

			for _, player := range payAnte {
				player.balance -= g.ante
				newPot += g.ante
			}

			// if more than one winner, pot grows
			if len(winners) > 1 {
				newPot += g.pot
			} else {
				winningAmount = g.pot
				winners[0].balance += winningAmount
			}
		} else {
			winningAmount = g.pot
			winners[0].balance += winningAmount
			booted = append(booted, payAnte...)
			payAnte = make([]*Player, 0)
		}
	}

	g.result = &Result{
		Parent:        g.parentResult,
		PaidAnte:      payAnte,
		PaidPot:       payPot,
		Winners:       winners,
		Folded:        g.getFoldedPlayers(),
		Booted:        booted,
		WinningAmount: winningAmount,
		Ante:          g.ante,
		OldPot:        g.pot,
		NewPot:        newPot,
		logger:        g.logger,
		logChan:       g.logChan,
		playerOrder:   g.playerOrder,
		idToPlayer:    g.idToPlayer,
	}

	return nil
}

// isTradeInRound returns true if the trade in round is in progress
func (g *Game) isTradeInRound() bool {
	return g.roundNo == 0
}

func (g *Game) getFoldedPlayers() []*Player {
	players := make([]*Player, 0, len(g.foldedPlayers))
	for player := range g.foldedPlayers {
		players = append(players, player)
	}

	return players
}

// playerDidPlayCard plays the card for the player
func (g *Game) playerDidPlayCard(player *Player, card *deck.Card) error {
	if g.isTradeInRound() {
		return ErrTradeInRoundInProgress
	}

	if g.isRoundOver() {
		return ErrRoundIsOver
	}

	if !g.isPlayersTurn(player) {
		return ErrIsNotPlayersTurn
	}

	playedCard := &playedCard{
		player: player,
		card:   card,
	}

	found := false
	for _, c := range player.hand {
		if c.String() == card.String() {
			found = true
		}
	}

	if !found {
		return ErrCardNotInPlayersHand
	}

	if err := g.canPlayerPlayCard(player, card); err != nil {
		return err
	}

	// this should not happen as we already checked these cases.
	// just one more safeguard just in case
	if err := player.playerDidPlayCard(card); err != nil {
		panic(err)
	}

	if g.winningCardPlayed == nil {
		g.winningCardPlayed = playedCard
	} else {
		wcRank := g.winningCardPlayed.card.Rank
		wcSuit := g.winningCardPlayed.card.Suit
		tcSuit := g.trumpCard.Suit
		if (card.Rank > wcRank && card.Suit == wcSuit) ||
			(card.Suit == tcSuit && wcSuit != tcSuit) {
			g.winningCardPlayed = playedCard
		}
	}

	g.cardsPlayed = append(g.cardsPlayed, playedCard)

	if g.shouldCalculateRoundWinner() {
		return g.calculateRoundWinner()
	}

	return nil
}

// canPlayerPlayCard returns nil if the player can play the specified card
// This method does not check if it's actually the player's turn or not
func (g *Game) canPlayerPlayCard(player *Player, card *deck.Card) error {
	tcSuit := g.trumpCard.Suit
	if len(g.cardsPlayed) != 0 {
		wcSuit := g.winningCardPlayed.card.Suit
		wcRank := g.winningCardPlayed.card.Rank

		leadCard := g.cardsPlayed[0].card
		if card.Suit == leadCard.Suit {
			if card.Rank < wcRank && card.Suit == wcSuit {
				for _, c := range player.hand {
					if c.Rank > wcRank && c.Suit == wcSuit {
						return ErrPlayToWinOnSuit
					}
				}
			}
		} else if card.Suit == tcSuit {
			for _, c := range player.hand {
				if c.Suit == leadCard.Suit {
					return ErrPlayOnSuit
				}
			}

			if card.Rank < wcRank && wcSuit == tcSuit {
				for _, c := range player.hand {
					if c.Suit == tcSuit && c.Rank > wcRank {
						return ErrPlayToWinOnTrump
					}
				}
			}
		} else {
			for _, c := range player.hand {
				if c.Suit == leadCard.Suit {
					return ErrPlayOnSuit
				}
			}

			for _, c := range player.hand {
				if c.Suit == tcSuit {
					return ErrPlayTrump
				}
			}
		}
	}
	return nil
}

// isGameOver returns true if the game is over
func (g *Game) isGameOver() bool {
	return g.result != nil
}

// isRoundOver returns true if the round is over
func (g *Game) isRoundOver() bool {
	if g.isTradeInRound() {
		return len(g.playerDiscards) >= len(g.playerOrder)
	}

	return len(g.cardsPlayed) >= len(g.playerOrder)
}

// calculateRoundWinner is called after the last card in a round has been played
func (g *Game) calculateRoundWinner() error {
	if !g.isRoundOver() {
		return ErrRoundNotOver
	}

	if g.canGameEnd() {
		return ErrGameIsOver
	}

	if g.roundWinnerCalculated {
		panic("calculateRoundWinner() already called")
	}

	g.roundWinnerCalculated = true

	winningPlayer := g.winningCardPlayed.player
	winningPlayer.WonRound()

	return nil
}

// nextRound puts the game in a state for the next round
func (g *Game) nextRound() error {
	if !g.roundWinnerCalculated {
		return ErrRoundNotOver
	}

	g.sendLogMessages(newLogMessage(g.winningCardPlayed.player.PlayerID, nil, "{} won the trick"))
	g.roundWinnerCalculated = false
	g.winningCardPlayed = nil
	g.cardsPlayed = []*playedCard{}
	g.roundNo++

	for player := range g.playerOrder {
		player.NewRound()
	}

	if g.shouldBuildResults() {
		return g.buildResults()
	}

	return nil
}

func (g *Game) shouldCalculateRoundWinner() bool {
	return len(g.cardsPlayed) == len(g.playerOrder)
}

func (g *Game) shouldBuildResults() bool {
	return g.roundNo == 6
}

// getCurrentTurn returns the current player
// If it's the end of the round, return nil
func (g *Game) getCurrentTurn() *Player {
	if g.isRoundOver() {
		return nil
	}

	var index int
	if g.isTradeInRound() {
		index = len(g.playerDiscards)
	} else {
		// -1 because round 0 is the trade-in-round
		index = (len(g.cardsPlayed) + g.roundNo - 1) % len(g.playerOrder)
	}

	for player, i := range g.playerOrder {
		if i == index {
			return player
		}
	}

	panic("could not get current turn")
}

// isPlayersTurn returns true if the player can play a card
func (g *Game) isPlayersTurn(p *Player) bool {
	return g.getCurrentTurn() == p
}

func (g *Game) sendLogMessages(msg ...*playable.LogMessage) {
	if g.logChan != nil {
		g.logChan <- msg
	}
}

func newLogMessage(playerID int64, card *deck.Card, format string, a ...interface{}) *playable.LogMessage {
	var cards []*deck.Card
	if card != nil {
		cards = append(cards, card)
	}
	return &playable.LogMessage{
		UUID:      uuid.New().String(),
		PlayerIDs: []int64{playerID},
		Cards:     cards,
		Message:   fmt.Sprintf(format, a...),
		Time:      time.Now(),
	}
}

func newLogMessageWithPlayers(players []*Player, format string, a ...interface{}) *playable.LogMessage {
	ids := make([]int64, len(players))
	for i, player := range players {
		ids[i] = player.PlayerID
	}

	return &playable.LogMessage{
		UUID:      uuid.New().String(),
		PlayerIDs: ids,
		Message:   fmt.Sprintf(format, a...),
		Time:      time.Now(),
	}
}

// NameFromOptions returns the name of the game based on the options
func NameFromOptions(opts Options) string {
	name := "Bourré"
	if opts.FiveSuit {
		name += " (Five Suit)"
	}

	return name
}
