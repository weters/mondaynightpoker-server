package guts

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Phase represents the current phase of the game
type Phase int

const (
	// PhaseDealing is when cards are being dealt
	PhaseDealing Phase = iota
	// PhaseDeclaration is when players decide in/out
	PhaseDeclaration
	// PhaseTradeIn is when players who went In can trade cards
	PhaseTradeIn
	// PhaseShowdown is when hands are revealed and compared
	PhaseShowdown
	// PhaseRoundEnd is the end of a round before the next
	PhaseRoundEnd
	// PhaseGameOver is when the game has ended
	PhaseGameOver
)

// ShowdownResult contains the results of a showdown
type ShowdownResult struct {
	Winners      []*Participant
	Losers       []*Participant
	PlayersIn    []*Participant
	WinningHand  HandResult
	PotWon       int
	PenaltyPaid  int
	NextPot      int
	AllFolded    bool
	SingleWinner bool
	DeckHand     []*deck.Card // Deck's cards when Bloody Guts triggered
	DeckWon      bool         // True if deck beat the player in Bloody Guts
}

// Game is a game of 2-card guts
type Game struct {
	options         Options
	deck            *deck.Deck
	participants    []*Participant
	idToParticipant map[int64]*Participant

	pot         int
	phase       Phase
	roundNumber int

	// Simultaneous declaration tracking
	pendingDecisions map[int64]bool // Who hasn't decided yet
	decisions        map[int64]bool // true=In, false=Out

	showdownResult    *ShowdownResult
	deckHand          []*deck.Card // Stores deck's hand when Bloody Guts is triggered
	deckCardsRevealed int          // Track how many deck cards have been shown
	bloodyGutsPlayer  int64        // Track player in bloody guts showdown

	// Trade phase tracking
	discards           []*deck.Card   // Discarded cards for reshuffling
	tradersIn          []*Participant // Players who went In (in participant order)
	currentTraderIndex int            // Whose turn to trade
	tradesMade         map[int64]int  // Track cards traded per player

	done bool

	logger  logrus.FieldLogger
	logChan chan []*playable.LogMessage

	pendingDealerAction *pendingDealerAction
}

// Interval determines how often Tick() should be called
func (g *Game) Interval() time.Duration {
	return time.Second
}

// Tick will check the state of the game and possibly move the state along
func (g *Game) Tick() (bool, error) {
	if g.done {
		return false, nil
	}

	if g.pendingDealerAction != nil {
		if time.Now().After(g.pendingDealerAction.ExecuteAfter) {
			action := g.pendingDealerAction.Action
			// Clear BEFORE executing so actions can schedule new ones
			g.pendingDealerAction = nil

			switch action {
			case dealerActionShowdown:
				g.calculateShowdown()
			case dealerActionRevealDeckCard:
				g.revealNextDeckCard()
			case dealerActionResolveBloodyGuts:
				g.resolveBloodyGuts()
			case dealerActionNextRound:
				if err := g.nextRound(); err != nil {
					logrus.WithError(err).Error("could not go to the next round")
				}
			case dealerActionEndGame:
				g.phase = PhaseGameOver
				g.done = true
			case dealerActionNextTrader:
				g.notifyNextTrader()
			default:
				panic(fmt.Sprintf("unknown dealer action: %d", action))
			}

			return true, nil
		}

		return false, nil
	}

	return false, nil
}

// Name returns "guts"
func (g *Game) Name() string {
	return "guts"
}

// LogChan returns a channel for sending log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}

// Action performs an action
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	if g.phase == PhaseGameOver {
		return nil, false, ErrGameIsOver
	}

	_, ok := g.idToParticipant[playerID]
	if !ok {
		return nil, false, ErrPlayerNotFound
	}

	switch message.Action {
	case "decide":
		goIn, ok := message.AdditionalData.GetBool("in")
		if !ok {
			return nil, false, errors.New("missing 'in' parameter")
		}

		if err := g.submitDecision(playerID, goIn); err != nil {
			return nil, false, err
		}

		return playable.OK(), true, nil
	case "trade":
		cardsData := getStringSlice(message.AdditionalData, "cards")

		cards, err := parseCards(cardsData)
		if err != nil {
			return nil, false, err
		}

		if err := g.submitTrade(playerID, cards); err != nil {
			return nil, false, err
		}

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

	adjustments := make(map[int64]int)
	for _, p := range g.participants {
		adjustments[p.PlayerID] = p.balance
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: adjustments,
		Log:                g.showdownResult,
	}, true
}

// NewGame returns a new guts game
func NewGame(logger logrus.FieldLogger, playerIDs []int64, opts Options) (*Game, error) {
	if len(playerIDs) < 2 || len(playerIDs) > 10 {
		return nil, PlayerCountError{
			Min: 2,
			Max: 10,
			Got: len(playerIDs),
		}
	}

	participants := make([]*Participant, len(playerIDs))
	idToParticipant := make(map[int64]*Participant)

	for i, pid := range playerIDs {
		p := NewParticipant(pid)
		participants[i] = p
		idToParticipant[pid] = p
	}

	d := deck.New()
	d.Shuffle()

	pot := 0
	messages := make([]*playable.LogMessage, 0)

	for _, p := range participants {
		pot += opts.Ante
		p.balance -= opts.Ante
		messages = append(messages, newLogMessage(p.PlayerID, "{} paid the ${%d} ante", opts.Ante))
	}

	g := &Game{
		options:          opts,
		deck:             d,
		participants:     participants,
		idToParticipant:  idToParticipant,
		pot:              pot,
		phase:            PhaseDealing,
		roundNumber:      1,
		pendingDecisions: make(map[int64]bool),
		decisions:        make(map[int64]bool),
		logger:           logger,
		logChan:          make(chan []*playable.LogMessage, 256),
	}

	messages = append(messages, newLogMessage(0, "New game of %s started with a pot of ${%d}", NameFromOptions(opts), pot))
	g.sendLogMessages(messages...)

	return g, nil
}

// Deal will deal cards to each participant
func (g *Game) Deal() error {
	if len(g.participants) < 2 {
		return ErrNotEnoughPlayers
	}

	// Clear hands and reset for new round
	for _, p := range g.participants {
		p.ClearHand()
	}

	// Deal cards to each player
	cardCount := g.options.CardCount
	if cardCount < 2 || cardCount > 3 {
		cardCount = 2
	}
	for i := 0; i < cardCount; i++ {
		for _, p := range g.participants {
			card, err := g.deck.Draw()
			if err != nil {
				return err
			}
			p.AddCard(card)
		}
	}

	// Initialize pending decisions
	g.pendingDecisions = make(map[int64]bool)
	g.decisions = make(map[int64]bool)
	for _, p := range g.participants {
		g.pendingDecisions[p.PlayerID] = true
	}

	g.phase = PhaseDeclaration
	g.sendLogMessages(newLogMessage(0, "Round %d: Cards dealt, declare In or Out", g.roundNumber))

	return nil
}

// submitDecision records a player's in/out decision
func (g *Game) submitDecision(playerID int64, goIn bool) error {
	if g.phase != PhaseDeclaration {
		return ErrNotInDeclarationPhase
	}

	if !g.pendingDecisions[playerID] {
		return ErrAlreadyDecided
	}

	g.decisions[playerID] = goIn
	delete(g.pendingDecisions, playerID)

	// Log that player has decided (without revealing the decision)
	g.sendLogMessages(newLogMessage(playerID, "{} has decided"))

	// Only reveal when ALL have decided
	if len(g.pendingDecisions) == 0 {
		// Reveal all decisions
		for _, p := range g.participants {
			decision := "Out"
			if g.decisions[p.PlayerID] {
				decision = "In"
			}
			g.sendLogMessages(newLogMessage(p.PlayerID, "{} was %s", decision))
		}

		// Count players who went In
		playersInCount := 0
		for _, goIn := range g.decisions {
			if goIn {
				playersInCount++
			}
		}

		// Start trade phase if AllowTrades enabled AND:
		// - 2+ players went In, OR
		// - 1 player went In AND BloodyGuts enabled (trade before facing deck)
		shouldTrade := g.options.AllowTrades && (playersInCount >= 2 || (playersInCount == 1 && g.options.BloodyGuts))
		if shouldTrade {
			g.startTradeInPhase()
		} else {
			g.pendingDealerAction = &pendingDealerAction{
				Action:       dealerActionShowdown,
				ExecuteAfter: time.Now().Add(time.Second),
			}
		}
	}

	return nil
}

// calculateShowdown determines winners and losers
func (g *Game) calculateShowdown() {
	g.phase = PhaseShowdown

	// Find players who went in
	playersIn := make([]*Participant, 0)
	for _, p := range g.participants {
		if g.decisions[p.PlayerID] {
			playersIn = append(playersIn, p)
		}
	}

	result := &ShowdownResult{
		PlayersIn: playersIn,
	}

	// Case 1: No one goes in - everyone re-antes
	if len(playersIn) == 0 {
		result.AllFolded = true
		g.showdownResult = result
		g.sendLogMessages(newLogMessage(0, "No one went in! Everyone re-antes."))

		// Schedule next round
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionNextRound,
			ExecuteAfter: time.Now().Add(time.Second * 5),
		}
		return
	}

	// Case 2: Only one person goes in
	if len(playersIn) == 1 {
		player := playersIn[0]

		// Bloody Guts: player must beat the deck
		if g.options.BloodyGuts {
			g.handleBloodyGutsShowdown(player, result)
			return
		}

		// Regular mode: player wins automatically
		player.balance += g.pot
		result.Winners = []*Participant{player}
		result.WinningHand = AnalyzeHand(player.hand)
		result.PotWon = g.pot
		result.SingleWinner = true
		g.showdownResult = result

		g.sendLogMessages(newLogMessageWithCards(player.PlayerID, player.hand,
			"{} wins ${%d} with %s", g.pot, HandTypeName(result.WinningHand.Type)))

		// Game ends (phase stays at PhaseShowdown so winning hand is visible)
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionEndGame,
			ExecuteAfter: time.Now().Add(time.Second * 5),
		}
		return
	}

	// Case 3: Multiple players go in - compare hands
	// Find the best hand(s)
	var bestStrength int
	winners := make([]*Participant, 0)

	for _, p := range playersIn {
		handResult := AnalyzeHand(p.hand)
		if handResult.Strength > bestStrength {
			bestStrength = handResult.Strength
			winners = []*Participant{p}
		} else if handResult.Strength == bestStrength {
			winners = append(winners, p)
		}
	}

	// Determine losers (in players who didn't win)
	losers := make([]*Participant, 0)
	for _, p := range playersIn {
		isWinner := false
		for _, w := range winners {
			if w.PlayerID == p.PlayerID {
				isWinner = true
				break
			}
		}
		if !isWinner {
			losers = append(losers, p)
		}
	}

	result.Winners = winners
	result.Losers = losers
	result.WinningHand = AnalyzeHand(winners[0].hand)
	result.PotWon = g.pot

	// Distribute pot to winners
	winPerPerson := g.pot / len(winners)
	remainder := g.pot % len(winners)
	for i, w := range winners {
		winAmount := winPerPerson
		if i < remainder {
			winAmount++ // Distribute remainder
		}
		w.balance += winAmount
	}

	// Calculate penalty (capped at maxOwed)
	penalty := g.calculatePenalty()
	result.PenaltyPaid = penalty

	// Losers pay penalty into next pot
	nextPot := 0
	for _, loser := range losers {
		loser.balance -= penalty
		nextPot += penalty
	}
	result.NextPot = nextPot

	g.showdownResult = result

	// Log results
	if len(winners) == 1 {
		g.sendLogMessages(newLogMessageWithCards(winners[0].PlayerID, winners[0].hand,
			"{} wins ${%d} with %s", g.pot, HandTypeName(result.WinningHand.Type)))
	} else {
		playerIDs := make([]int64, len(winners))
		for i, w := range winners {
			playerIDs[i] = w.PlayerID
		}
		g.sendLogMessages(newLogMessageWithPlayers(playerIDs, "{} split the pot of ${%d}", g.pot))
	}

	for _, loser := range losers {
		g.sendLogMessages(newLogMessage(loser.PlayerID, "{} pays penalty of ${%d}", penalty))
	}

	// If there are losers who paid penalties, continue the game
	if nextPot > 0 {
		g.pot = nextPot
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionNextRound,
			ExecuteAfter: time.Now().Add(time.Second * 5),
		}
	} else {
		// No penalties paid (everyone who was in won), game ends
		// Phase stays at PhaseShowdown so winning hand is visible
		g.sendLogMessages(newLogMessage(0, "The game ends"))
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionEndGame,
			ExecuteAfter: time.Now().Add(time.Second * 5),
		}
	}
}

// calculatePenalty returns the penalty amount (pot capped at maxOwed)
func (g *Game) calculatePenalty() int {
	if g.pot > g.options.MaxOwed {
		return g.options.MaxOwed
	}
	return g.pot
}

// handleBloodyGutsShowdown handles the case where a single player must beat the deck
func (g *Game) handleBloodyGutsShowdown(player *Participant, result *ShowdownResult) {
	// Draw cards from deck (but don't reveal yet)
	cardCount := g.options.CardCount
	if cardCount < 2 || cardCount > 3 {
		cardCount = 2
	}
	drawCount := cardCount
	if g.options.AllowTrades {
		drawCount++
	}

	g.deckHand = make([]*deck.Card, drawCount)
	for i := 0; i < drawCount; i++ {
		card, err := g.deck.Draw()
		if err != nil {
			// This shouldn't happen in normal play
			g.logger.WithError(err).Error("failed to draw deck card for Bloody Guts")
			return
		}
		g.deckHand[i] = card
	}

	// Set up for staged reveal
	g.deckCardsRevealed = 0
	g.bloodyGutsPlayer = player.PlayerID
	g.showdownResult = result

	g.sendLogMessages(newLogMessage(player.PlayerID, "{} faces the deck..."))

	// Schedule first card reveal (2 second delay)
	g.pendingDealerAction = &pendingDealerAction{
		Action:       dealerActionRevealDeckCard,
		ExecuteAfter: time.Now().Add(time.Second * 2),
	}
}

// revealNextDeckCard reveals the next deck card in a Bloody Guts showdown
func (g *Game) revealNextDeckCard() {
	g.deckCardsRevealed++

	card := g.deckHand[g.deckCardsRevealed-1]
	g.sendLogMessages(newLogMessage(0, "Deck reveals: %s", card.String()))

	if g.deckCardsRevealed < len(g.deckHand) {
		// More cards to reveal - schedule next reveal (2 seconds)
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionRevealDeckCard,
			ExecuteAfter: time.Now().Add(time.Second * 2),
		}
	} else {
		// All cards revealed - resolve the showdown (1 second delay)
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionResolveBloodyGuts,
			ExecuteAfter: time.Now().Add(time.Second),
		}
	}
}

// resolveBloodyGuts determines the winner after all deck cards are revealed
func (g *Game) resolveBloodyGuts() {
	player := g.idToParticipant[g.bloodyGutsPlayer]
	result := g.showdownResult

	playerHand := AnalyzeHand(player.hand)
	bestDeckCards := BestHand(g.deckHand, g.options.CardCount)
	deckHandResult := AnalyzeHand(bestDeckCards)

	result.DeckHand = g.deckHand

	// Player must strictly beat the deck (deck wins on ties)
	if playerHand.Strength > deckHandResult.Strength {
		// Player wins
		player.balance += g.pot
		result.Winners = []*Participant{player}
		result.WinningHand = playerHand
		result.PotWon = g.pot
		result.SingleWinner = true
		result.DeckWon = false

		g.sendLogMessages(
			newLogMessageWithCards(player.PlayerID, player.hand,
				"{} beats the deck with %s and wins ${%d}", HandTypeName(playerHand.Type), g.pot),
		)

		// Game ends (phase stays at PhaseShowdown so winning hand is visible)
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionEndGame,
			ExecuteAfter: time.Now().Add(time.Second * 5),
		}
	} else {
		// Deck wins (including ties)
		result.Losers = []*Participant{player}
		result.DeckWon = true

		penalty := g.calculatePenalty()
		result.PenaltyPaid = penalty

		player.balance -= penalty
		result.NextPot = g.pot + penalty
		g.pot += penalty

		g.sendLogMessages(
			newLogMessage(player.PlayerID, "The deck wins with %s! {} pays penalty of ${%d}",
				HandTypeName(deckHandResult.Type), penalty),
		)

		// Continue to next round
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionNextRound,
			ExecuteAfter: time.Now().Add(time.Second * 5),
		}
	}
}

// nextRound starts a new round
func (g *Game) nextRound() error {
	// If everyone folded, re-ante
	if g.showdownResult != nil && g.showdownResult.AllFolded {
		for _, p := range g.participants {
			p.balance -= g.options.Ante
			g.pot += g.options.Ante
		}
		g.sendLogMessages(newLogMessage(0, "Everyone re-anted. Pot is now ${%d}", g.pot))
	}

	g.roundNumber++
	g.showdownResult = nil
	g.deckHand = nil
	g.deckCardsRevealed = 0
	g.bloodyGutsPlayer = 0

	// Clear trade state
	g.discards = nil
	g.tradersIn = nil
	g.currentTraderIndex = 0
	g.tradesMade = nil
	for _, p := range g.participants {
		p.traded = 0
	}

	// Reshuffle deck
	g.deck = deck.New()
	g.deck.Shuffle()

	// Deal new round
	return g.Deal()
}

func (g *Game) sendLogMessages(msg ...*playable.LogMessage) {
	if g.logChan != nil {
		g.logChan <- msg
	}
}

func newLogMessage(playerID int64, format string, a ...interface{}) *playable.LogMessage {
	return &playable.LogMessage{
		UUID:      uuid.New().String(),
		PlayerIDs: []int64{playerID},
		Message:   fmt.Sprintf(format, a...),
		Time:      time.Now(),
	}
}

// parseCards safely parses card strings, recovering from panics
func parseCards(cardStrings []string) (cards []*deck.Card, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid card format: %v", r)
			cards = nil
		}
	}()

	cards = make([]*deck.Card, len(cardStrings))
	for i, cardStr := range cardStrings {
		cards[i] = deck.CardFromString(cardStr)
	}
	return cards, nil
}

// getStringSlice extracts a string slice from AdditionalData
func getStringSlice(data playable.AdditionalData, key string) []string {
	val, ok := data[key]
	if !ok {
		return []string{}
	}

	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return []string{}
	}
}

func newLogMessageWithPlayers(playerIDs []int64, format string, a ...interface{}) *playable.LogMessage {
	return &playable.LogMessage{
		UUID:      uuid.New().String(),
		PlayerIDs: playerIDs,
		Message:   fmt.Sprintf(format, a...),
		Time:      time.Now(),
	}
}

func newLogMessageWithCards(playerID int64, cards []*deck.Card, format string, a ...interface{}) *playable.LogMessage {
	return &playable.LogMessage{
		UUID:      uuid.New().String(),
		PlayerIDs: []int64{playerID},
		Cards:     cards,
		Message:   fmt.Sprintf(format, a...),
		Time:      time.Now(),
	}
}

// startTradeInPhase initializes the trade phase after all decisions are revealed
func (g *Game) startTradeInPhase() {
	g.phase = PhaseTradeIn

	// Build list of players who went In, in participant order
	g.tradersIn = make([]*Participant, 0)
	for _, p := range g.participants {
		if g.decisions[p.PlayerID] {
			g.tradersIn = append(g.tradersIn, p)
		}
	}

	g.currentTraderIndex = 0
	g.tradesMade = make(map[int64]int)
	g.discards = make([]*deck.Card, 0)

	g.sendLogMessages(newLogMessage(0, "Trade round begins"))

	// Notify first trader
	g.pendingDealerAction = &pendingDealerAction{
		Action:       dealerActionNextTrader,
		ExecuteAfter: time.Now().Add(500 * time.Millisecond),
	}
}

// notifyNextTrader sends a log message that it's the current trader's turn
func (g *Game) notifyNextTrader() {
	if g.currentTraderIndex >= len(g.tradersIn) {
		// All done trading, proceed to showdown
		g.sendLogMessages(newLogMessage(0, "Trading complete"))
		g.pendingDealerAction = &pendingDealerAction{
			Action:       dealerActionShowdown,
			ExecuteAfter: time.Now().Add(time.Second),
		}
		return
	}

	trader := g.tradersIn[g.currentTraderIndex]
	g.sendLogMessages(newLogMessage(trader.PlayerID, "{}'s turn to trade"))
}

// getCurrentTrader returns the current trader or nil if not in trade phase
func (g *Game) getCurrentTrader() *Participant {
	if g.phase != PhaseTradeIn || g.currentTraderIndex >= len(g.tradersIn) {
		return nil
	}
	return g.tradersIn[g.currentTraderIndex]
}

// submitTrade handles a player's trade action
func (g *Game) submitTrade(playerID int64, cards []*deck.Card) error {
	if g.phase != PhaseTradeIn {
		return ErrNotInTradePhase
	}

	currentTrader := g.getCurrentTrader()
	if currentTrader == nil || currentTrader.PlayerID != playerID {
		return ErrNotYourTurnToTrade
	}

	cardCount := g.options.CardCount
	if cardCount < 2 || cardCount > 3 {
		cardCount = 2
	}

	if len(cards) > cardCount {
		return ErrInvalidTradeCount
	}

	// Validate all cards are in hand
	for _, card := range cards {
		if !currentTrader.hasCard(card) {
			return ErrCardNotInHand
		}
	}

	// Remove cards from hand and add to discards
	for _, card := range cards {
		currentTrader.removeCard(card)
		g.discards = append(g.discards, card)
	}

	// Draw replacement cards
	for i := 0; i < len(cards); i++ {
		// Check if deck is empty and reshuffle if needed
		if len(g.deck.Cards) == 0 {
			if len(g.discards) == 0 {
				// This shouldn't happen in normal play
				g.logger.Error("deck and discards both empty during trade")
				break
			}
			g.deck.ShuffleDiscards(g.discards)
			g.discards = make([]*deck.Card, 0)
			g.sendLogMessages(newLogMessage(0, "Deck reshuffled"))
		}

		card, err := g.deck.Draw()
		if err != nil {
			g.logger.WithError(err).Error("failed to draw card during trade")
			break
		}
		currentTrader.AddCard(card)
	}

	g.tradesMade[playerID] = len(cards)
	currentTrader.traded = len(cards)

	// Log the trade
	if len(cards) == 0 {
		g.sendLogMessages(newLogMessage(playerID, "{} stands pat"))
	} else if len(cards) == 1 {
		g.sendLogMessages(newLogMessage(playerID, "{} trades 1 card"))
	} else {
		g.sendLogMessages(newLogMessage(playerID, "{} trades %d cards", len(cards)))
	}

	// Advance to next trader
	g.advanceTrader()

	return nil
}

// advanceTrader moves to the next trader or proceeds to showdown
func (g *Game) advanceTrader() {
	g.currentTraderIndex++

	// Schedule next trader notification (or showdown if done)
	g.pendingDealerAction = &pendingDealerAction{
		Action:       dealerActionNextTrader,
		ExecuteAfter: time.Now().Add(500 * time.Millisecond),
	}
}

// NameFromOptions returns the name of the game based on options
func NameFromOptions(opts Options) string {
	prefix := ""
	if opts.BloodyGuts {
		prefix = "Bloody "
	}
	suffix := ""
	if opts.AllowTrades {
		suffix = " with Trades"
	}
	if opts.CardCount == 3 {
		return prefix + "3-Card Guts" + suffix
	}
	return prefix + "2-Card Guts" + suffix
}
