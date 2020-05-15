package bourre

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"
)

const playersLimit = 8

// ErrLastPlayerMustPlay prevents the last player left from folding
var ErrLastPlayerMustPlay = errors.New("everyone else folded, so you must play")

// ErrNotEnoughActivePlayers is an error when there are not at least two active players
var ErrNotEnoughActivePlayers = errors.New("need at least two players to continue")

// ErrTradeInRoundInProgress happens if a player tries to play a card before the trade-in round is complete
var ErrTradeInRoundInProgress = errors.New("the trade-in round is not complete")

// ErrTradeInRoundIsOver happens if trade are attempted during the game play
var ErrTradeInRoundIsOver = errors.New("the trade-in round is over")

// PlayerCountError is an error on the number of players in the game
type PlayerCountError int

func (p PlayerCountError) Error() string {
	return fmt.Sprintf("expected 2–%d players, got %d", playersLimit, p)
}

// Game is a game of bourré
type Game struct {
	table         string
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

	logChan chan []*playable.LogMessage

	// done will be set after the game is over and the user's have stated they want to proceed
	done bool
}

// Name returns "bourre"
func (g *Game) Name() string {
	return "bourre"
}

// LogChan returns a channel for sending log messages
func (g *Game) LogChan() chan []*playable.LogMessage {
	return g.logChan
}

// Action performs an action
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	player, ok := g.idToPlayer[playerID]
	if !ok {
		return nil, false, errors.New("player not found with that ID")
	}

	log := logrus.WithFields(logrus.Fields{
		"game":     "bourre",
		"playerID": playerID,
		"table":    g.table,
	})

	switch message.Action {
	case "discard":
		log.WithField("cards", message.Cards).Debug("player discards")
		if err := g.PlayerDiscards(player, message.Cards); err != nil {
			return nil, false, err
		}

		if message.Cards == nil {
			g.sendLogMessages(newLogMessage(player.PlayerID, "{} folded"))
		} else {
			g.sendLogMessages(newLogMessage(player.PlayerID, "{} discarded %d", len(message.Cards)))
		}

		return playable.OK(), true, nil
	case "replaceDiscards":
		log.Debug("replaceDiscards triggered")
		if err := g.ReplaceDiscards(); err != nil {
			return nil, false, err
		}

		g.sendLogMessages(newLogMessage(0, "Dealer replaced discards"))
		return playable.OK(), true, nil
	case "playCard":
		if len(message.Cards) != 1 {
			return nil, false, fmt.Errorf("expected to get 1 card, got %d", len(message.Cards))
		}

		log.WithField("card", message.Cards[0]).Debug("play card")
		if err := g.PlayCard(player, message.Cards[0]); err != nil {
			return nil, false, err
		}

		g.sendLogMessages(newLogMessage(player.PlayerID, "{} played the %s", message.Cards[0]))
		return playable.OK(), true, nil
	case "nextRound":
		log.Debug("nextRound triggered")
		if err := g.NextRound(); err != nil {
			return nil, false, err
		}

		g.sendLogMessages(newLogMessage(0, "Next round started"))
		return playable.OK(), true, nil
	case "done":
		res := g.result
		if res == nil {
			return nil, false, errors.New("done cannot be called yet")
		}

		messages := make([]*playable.LogMessage, 0)

		if res.WinningAmount > 0 {
			messages = append(messages, newLogMessage(res.Winners[0].PlayerID, "{} won %d¢", res.WinningAmount))
		} else {
			messages = append(messages, newLogMessageWithPlayers(res.Winners, "{} tied for most tricks"))
		}

		if len(res.PaidPot) > 0 {
			messages = append(messages, newLogMessageWithPlayers(res.PaidPot, "{} pays the pot of %d¢", res.OldPot))
		}

		if len(res.PaidAnte) > 0 {
			messages = append(messages, newLogMessageWithPlayers(res.PaidAnte, "{} pays the ante of %d¢", res.Ante))
		}

		if len(res.Booted) > 0 {
			messages = append(messages, newLogMessageWithPlayers(res.Booted, "{} was booted"))
		}

		log.Debug("done triggered")
		if res.ShouldContinue() {
			log.Debug("new game created")
			game, err := res.NewGame()
			if err != nil {
				return nil, false, err
			}

			if err := game.Deal(); err != nil {
				return nil, false, err
			}

			messages = append(messages, newLogMessage(0, "Another game is required"))

			*g = *game
		} else {
			log.Debug("game is done")
			g.done = true

			messages = append(messages, newLogMessage(0, "The game ends"))
		}

		g.sendLogMessages(messages...)
		return playable.OK(), true, nil

	default:
		return nil, false, fmt.Errorf("unknown action: %s", message.Action)
	}
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
func NewGame(tableUUID string, playerIDs []int64, opts Options) (*Game, error) {
	idToPlayer := make(map[int64]*Player)
	players := make([]*Player, len(playerIDs))
	for i, pid := range playerIDs {
		players[i] = NewPlayer(pid)
		idToPlayer[pid] = players[i]
	}

	g, err := newGame(players, nil, opts)
	if err != nil {
		return nil, err
	}

	g.idToPlayer = idToPlayer
	g.table = tableUUID
	return g, nil
}

func newGame(players []*Player, foldedPlayers []*Player, opts Options) (*Game, error) {
	if len(players) < 2 || len(players) > playersLimit {
		return nil, PlayerCountError(len(players))
	}

	pot := opts.InitialPot

	messages := make([]*playable.LogMessage, 0)

	ids := make([]int64, 0, len(players))
	playerOrder := make(map[*Player]int)
	for order, player := range players {
		// if initial pot is > 0, that means we are working off of a previous game. In that case,
		// we already took care of the players who need to ante
		if opts.InitialPot == 0 {
			messages = append(messages, newLogMessage(player.PlayerID, "{} paid the %d¢ ante", opts.Ante))
			pot += opts.Ante
			player.balance -= opts.Ante
		}

		playerOrder[player] = order

		ids = append(ids, player.PlayerID)
	}

	d := deck.New()
	d.Shuffle(0)

	logrus.WithFields(logrus.Fields{
		"players": ids,
		"seed":    d.Seed(),
		"hash":    d.HashCode(),
	}).Info("new game of bourré started")

	foldedPlayersMap := make(map[*Player]bool)
	if foldedPlayers != nil {
		for _, player := range foldedPlayers {
			foldedPlayersMap[player] = true
		}
	}

	g := &Game{
		deck:           d,
		pot:            pot,
		ante:           opts.Ante,
		playerOrder:    playerOrder,
		foldedPlayers:  foldedPlayersMap,
		playerDiscards: make(map[*Player][]*deck.Card),
		logChan:        make(chan []*playable.LogMessage, 256),
	}

	messages = append(messages, newLogMessage(0, "New game of Bourré started with a pot of %d¢", pot))
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

	logrus.WithFields(logrus.Fields{
		"game":      "bourre",
		"table":     g.table,
		"trumpCard": trumpCard,
	}).Debug("trump card")
	
	g.sendLogMessages(newLogMessage(0, "The trump card is %s", trumpCard))

	g.trumpCard = trumpCard
	return nil
}

// ErrCannotDiscardTheSameCard is an error when user has the same card in the/1186 discard array
var ErrCannotDiscardTheSameCard = errors.New("you cannot discard the same card")

func (g *Game) maxDraw(player *Player) int {
	pos := g.playerOrder[player]
	if len(g.playerOrder) > 5 && pos < 5 {
		return 3
	}

	return 4
}

// PlayerDiscards determines which cards to discard
// If discards is nil, the player elects to fold
func (g *Game) PlayerDiscards(player *Player, discards []*deck.Card) error {
	if !g.IsTradeInRound() {
		return ErrTradeInRoundIsOver
	}

	if g.IsRoundOver() {
		return ErrRoundIsOver
	}

	if g.GetCurrentTurn() != player {
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

// ReplaceDiscards will replace the discarded cards with fresh ones
func (g *Game) ReplaceDiscards() error {
	if len(g.playerDiscards) != len(g.playerOrder) {
		return ErrRoundNotOver
	}

	discardPile := make([]*deck.Card, 0)
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
			if err := player.PlayCard(card); err != nil {
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

// ErrGameIsOver is an error when an action is attempted on an ended game
var ErrGameIsOver = errors.New("game is over")

// ErrGameNotOver is an error when someone tries to end the game and it's not over yet
var ErrGameNotOver = errors.New("game is not over")

// ErrGameIsImmutable happens when a change is attempted after the game has finalized
var ErrGameIsImmutable = errors.New("game is over and no more changes can be made")

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
		Ante:          g.ante,
		PaidAnte:      payAnte,
		PaidPot:       payPot,
		Winners:       winners,
		Booted:        booted,
		Folded:        g.getFoldedPlayers(),
		WinningAmount: winningAmount,
		OldPot:        g.pot,
		NewPot:        newPot,
		playerOrder:   g.playerOrder,
		idToPlayer:    g.idToPlayer,
		table:         g.table,
	}

	return nil
}

// IsTradeInRound returns true if the trade in round is in progress
func (g *Game) IsTradeInRound() bool {
	return g.roundNo == 0
}

func (g *Game) getFoldedPlayers() []*Player {
	players := make([]*Player, 0, len(g.foldedPlayers))
	for player := range g.foldedPlayers {
		players = append(players, player)
	}

	return players
}

type playedCard struct {
	card   *deck.Card
	player *Player
}

// ErrRoundNotOver is an error when the round is not over yet
var ErrRoundNotOver = errors.New("the round is not over")

// ErrRoundIsOver is an error when cards beyond the round are played
var ErrRoundIsOver = errors.New("the round is over")

// ErrIsNotPlayersTurn is returned when it's not the player's turn
var ErrIsNotPlayersTurn = errors.New("not player's turn")

// ErrCardNotInPlayersHand happens when the player tries to play a card they don't have
var ErrCardNotInPlayersHand = errors.New("card is not in player's hand")

// ErrPlayToWinOnSuit happens when the player doesn't play a winning on-suit card
var ErrPlayToWinOnSuit = errors.New("player has a higher on-suit card")

// ErrPlayToWinOnTrump happens when the player doesn't play a winning trump card
var ErrPlayToWinOnTrump = errors.New("player has a higher trump card")

// ErrPlayOnSuit happens when a player has a suit of the lead suit and plays an off-suit card
var ErrPlayOnSuit = errors.New("player has an on-suit card")

// ErrPlayTrump happens if a player has a trump card and tries to play a non-trump, non lead
var ErrPlayTrump = errors.New("player has a trump card")

// PlayCard plays the card for the player
func (g *Game) PlayCard(player *Player, card *deck.Card) error {
	if g.IsTradeInRound() {
		return ErrTradeInRoundInProgress
	}

	if g.IsRoundOver() {
		return ErrRoundIsOver
	}

	if !g.IsPlayersTurn(player) {
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

	// this should not happen as we already checked these cases.
	// just one more safeguard just in case
	if err := player.PlayCard(card); err != nil {
		panic(err)
	}

	if g.winningCardPlayed == nil {
		g.winningCardPlayed = playedCard
	} else {
		wcRank := g.winningCardPlayed.card.Rank
		wcSuit := g.winningCardPlayed.card.Suit
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

func (g *Game) shouldCalculateRoundWinner() bool {
	return len(g.cardsPlayed) == len(g.playerOrder)
}

// IsRoundOver returns true if the round is over
func (g *Game) IsRoundOver() bool {
	if g.IsTradeInRound() {
		return len(g.playerDiscards) >= len(g.playerOrder)
	}

	return len(g.cardsPlayed) >= len(g.playerOrder)
}

// calculateRoundWinner is called after the last card in a round has been played
func (g *Game) calculateRoundWinner() error {
	if !g.IsRoundOver() {
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

// NextRound puts the game in a state for the next round
func (g *Game) NextRound() error {
	if !g.roundWinnerCalculated {
		return ErrRoundNotOver
	}

	g.sendLogMessages(newLogMessage(g.winningCardPlayed.player.PlayerID, "{} won the trick with the %s", g.winningCardPlayed.card))
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

func (g *Game) shouldBuildResults() bool {
	return g.roundNo == 6
}

// GetCurrentTurn returns the current player
// If it's the end of the round, return nil
func (g *Game) GetCurrentTurn() *Player {
	if g.IsRoundOver() {
		return nil
	}

	var index int
	if g.IsTradeInRound() {
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

// IsPlayersTurn returns true if the player can play a card
func (g *Game) IsPlayersTurn(p *Player) bool {
	return g.GetCurrentTurn() == p
}

func (g *Game) sendLogMessages(msg ...*playable.LogMessage) {
	g.logChan <- msg
}

func newLogMessage(playerID int64, format string, a ...interface{}) *playable.LogMessage {
	return &playable.LogMessage{
		PlayerIDs: []int64{playerID},
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
		PlayerIDs: ids,
		Message:   fmt.Sprintf(format, a...),
		Time:      time.Now(),
	}
}
