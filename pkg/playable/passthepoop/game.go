package passthepoop

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"strconv"
	"time"
)

// how long you must wait before you can hit next round after
// you end a round
const nextRoundDelay = time.Second * 2

// Game is an individual game of pass the poop
type Game struct {
	options         Options
	pot             int
	deck            *deck.Deck
	participants    []*Participant
	idToParticipant map[int64]*Participant

	logChan chan []*playable.LogMessage

	decisionIndex int
	pendingTrade  bool // was the decision to swap the card

	// did the dealer decide to go to the deck?
	dealerWillGoToDeck bool
	// prevent deal() from being called multiple times
	dealtCards bool

	// lastGameAction keeps track of the last successful game action
	lastGameAction *GameActionDetails

	// loserGroups will only be present after
	// EndRound() is called, and cleared when nextRound() is called
	loserGroups []*LoserGroup

	// balanceAdjustments will be nil until the end of game calculations have been made
	balanceAdjustments map[int64]int

	// endGameAck is when a player acknowledges the game is over and the UI can go to
	// game select screen
	endGameAck bool

	// nextRoundStartTime if not zero, cannot start next round unless
	// after now
	nextRoundStartTime time.Time
}

// random seed generator
// defined here for testing purposes
var seed = int64(0)

// NewGame returns a new game
func NewGame(tableUUID string, playerIDs []int64, options Options) (*Game, error) {
	if len(playerIDs) < 2 {
		return nil, errors.New("game requires at least two players")
	}

	if options.Ante <= 0 {
		return nil, errors.New("ante must be greater than 0")
	}

	if options.Lives <= 0 {
		return nil, errors.New("lives must be greater than 0")
	}

	d := deck.New()
	d.Shuffle(seed)
	logrus.WithField("seed", seed).WithField("deckSeed", d.Seed()).WithField("card", d.Cards[0]).Info("Shuffled")

	idToParticipants := make(map[int64]*Participant)
	participants := make([]*Participant, len(playerIDs))
	pot := 0
	for i, id := range playerIDs {
		pot += options.Ante
		participants[i] = &Participant{
			PlayerID: id,
			lives:    options.Lives,
			balance:  -1 * options.Ante,
		}
		idToParticipants[id] = participants[i]
	}

	g := &Game{
		deck:            d,
		pot:             pot,
		options:         options,
		participants:    participants,
		idToParticipant: idToParticipants,
		decisionIndex:   0,
		logChan:         make(chan []*playable.LogMessage, 256),
	}

	if err := g.deal(); err != nil {
		return nil, err
	}

	g.sendLogMessage(0, fmt.Sprintf("New game of Pass the Poop: %s Edition started", g.options.Edition.Name()))

	return g, nil
}

// ExecuteTurnForPlayer will perform a game action for the player
// A player can either stay or trade
// If trading, the player can either trade with the next player, or if they are the last player, they can go to the deck
// If trading with a player, and the next player has a King, they cannot trade
func (g *Game) ExecuteTurnForPlayer(playerID int64, gameAction GameAction) error {
	gameActionDetails := &GameActionDetails{
		GameAction:        gameAction,
		PlayerID:          playerID,
		SecondaryPlayerID: 0,
	}

	if err := g.executeTurnForPlayer(playerID, gameAction, gameActionDetails); err != nil {
		return err
	}

	switch gameAction {
	case ActionGoToDeck:
		g.sendLogMessage(playerID, "{} will go to the deck")
	case ActionTrade:
		g.sendLogMessage(playerID, "{} trades their card")
	case ActionAccept:
		g.sendLogMessage(playerID, "{} accepted the trade")
	case ActionFlipKing:
		p := g.idToParticipant[playerID]
		g.sendLogMessage(playerID, "{} revealed a King", p.card)
	case ActionDrawFromDeck:
		p := g.idToParticipant[playerID]
		g.sendLogMessage(playerID, "{} pulled a card from the deck", p.card)
	case ActionStay:
		g.sendLogMessage(playerID, "{} will stay")
	}

	// only save the details if it succeeded
	g.lastGameAction = gameActionDetails
	return nil
}

func (g *Game) executeTurnForPlayer(playerID int64, gameAction GameAction, gameActionDetails *GameActionDetails) error {
	if g.decisionIndex >= len(g.participants) {
		return errors.New("no more decisions can be made this round")
	}

	participant, ok := g.idToParticipant[playerID]
	if !ok {
		return fmt.Errorf("%d is not in this game", playerID)
	}

	if participant != g.getCurrentTurn() {
		return errors.New("you are not up")
	}

	switch gameAction {
	case ActionStay:
		if g.pendingTrade && participant.card.Rank == deck.King {
			return errors.New("you have to flip the King")
		}

		if g.pendingTrade {
			return errors.New("there is a pending trade you have to accept")
		}

		// do nothing
		g.decisionIndex++
		return nil
	case ActionGoToDeck:
		if !g.isDealersTurn() {
			return errors.New("only the dealer may go to the deck")
		}

		if participant.card.Rank == deck.King {
			return errors.New("dealer must stay with a King")
		}

		// going to the deck is a two-step process so we can first reveal the rest of the cards so the players
		// can see what the dealer needs. #Drama
		g.dealerWillGoToDeck = true

		g.flipAllCards()

		// do not advance decision index
		return nil
	case ActionDrawFromDeck:
		if !g.isDealersTurn() {
			return errors.New("only the dealer may draw from the deck")
		}

		if !g.dealerWillGoToDeck {
			return errors.New("you must first announce your intention to draw from the deck")
		}

		newCard, err := g.deck.Draw()
		if err != nil {
			return err
		}

		participant.card = newCard
		g.dealerWillGoToDeck = false
		g.decisionIndex++
		g.options.Edition.ParticipantWasPassed(participant, participant.card)
		return nil
	case ActionTrade:
		if participant.card.Rank == deck.King {
			return errors.New("you cannot trade a King")
		}

		if g.pendingTrade {
			return errors.New("there is a pending trade you have to accept")
		}

		if g.isDealersTurn() {
			return errors.New("the dealer can only go to the deck")
		}

		g.pendingTrade = true
		g.decisionIndex++
		gameActionDetails.SecondaryPlayerID = g.participants[g.decisionIndex].PlayerID
		return nil
	case ActionAccept:
		if !g.pendingTrade {
			return errors.New("there is no card to accept")
		}

		if participant.card.Rank == deck.King {
			return errors.New("you cannot accept the trade if you have a King")
		}

		g.pendingTrade = false

		prevParticipant := g.participants[g.decisionIndex-1]
		participant.card, prevParticipant.card = prevParticipant.card, participant.card

		g.options.Edition.ParticipantWasPassed(prevParticipant, prevParticipant.card)

		// do not increment the decision index, because the player still can make their own decision
		return nil
	case ActionFlipKing:
		if participant.card.Rank != deck.King {
			return errors.New("you do not have a King")
		}

		participant.isFlipped = true
		g.decisionIndex++
		g.pendingTrade = false
		return nil
	}

	return fmt.Errorf("not a valid game action")
}

// EndRound performs all necessary end of round actions
func (g *Game) EndRound() error {
	if g.getCurrentTurn() != nil {
		return errors.New("not all players have had a turn yet")
	}

	g.flipAllCards()

	loserGroups, err := g.options.Edition.EndRound(g.participants)
	if err != nil {
		if err == ErrMutualDestruction {
			logrus.WithError(err).Warn("round must be redone")
			g.loserGroups = make([]*LoserGroup, 0)
			return nil
		}

		return err
	}

	messages := make([]*playable.LogMessage, 0)
	g.loserGroups = loserGroups
	for _, group := range loserGroups {
		for _, loser := range group.RoundLosers {
			messages = append(messages, newLogMessage(loser.PlayerID, fmt.Sprintf("{} lost the round (-%d)", loser.LivesLost), loser.Card))
		}
	}
	g.logChan <- messages

	if !g.shouldContinue() {
		return g.endGame()
	}

	return nil
}

// endGame will calculate the end of game winner, make final balance adjustments
// Note: this method assumes we already checked that we can end the game
func (g *Game) endGame() error {
	if g.balanceAdjustments != nil {
		return errors.New("endGame() already called")
	}

	foundWinner := false
	adjustments := make(map[int64]int)
	for id, p := range g.idToParticipant {
		if p.lives > 0 {
			if foundWinner {
				return errors.New("too many winners found")
			}

			foundWinner = true
			p.balance += g.pot
		}

		adjustments[id] = p.balance
	}

	g.balanceAdjustments = adjustments
	return nil
}

// getCurrentTurn returns the participant who is currently making the decision
func (g *Game) getCurrentTurn() *Participant {
	if g.decisionIndex < len(g.participants) {
		return g.participants[g.decisionIndex]
	}

	return nil
}

// isDealersTurn returns true if the dealer is up
func (g *Game) isDealersTurn() bool {
	return g.decisionIndex+1 == len(g.participants)
}

// eliminateAndRotateParticipants removes eliminated players, and rotates the dealer button
func (g *Game) eliminateAndRotateParticipants() {
	newList := make([]*Participant, 0, len(g.participants))

	// this essentially does a shift and push (makes 1st position [index=0] into the dealer [index=n-1])
	// and remove any players no longer active
	for i := 1; i <= len(g.participants); i++ {
		pIndex := i % len(g.participants)
		participant := g.participants[pIndex]

		if participant.lives > 0 {
			newList = append(newList, participant)
		}
	}

	g.participants = newList
}

// shouldContinue returns true if there are still active participants left
// You should call this method after eliminateAndRotateParticipants()
func (g *Game) shouldContinue() bool {
	// XXX: may want to cache the results here to prevent repeated loop lookups
	active := 0
	for _, p := range g.participants {
		if p.lives > 0 {
			active++

			if active >= 2 {
				return true
			}
		}
	}

	return false
}

// nextRound will handle cleanup and set state for next round
// 1. Determine next dealer
// 2. Set next decision index
// Do not call nextRound() unless you know the game can continue
func (g *Game) nextRound() error {
	g.eliminateAndRotateParticipants()

	if len(g.participants) < 2 {
		return errors.New("not enough players for a new round")
	}

	g.sendLogMessage(0, "Next round started")

	g.dealtCards = false
	g.decisionIndex = 0
	g.dealerWillGoToDeck = false
	g.loserGroups = nil

	return g.deal()
}

func (g *Game) deal() error {
	if g.dealtCards {
		return errors.New("already dealt cards this round")
	}

	// +1 because dealer may go to the deck
	if !g.deck.CanDraw(len(g.participants) + 1) {
		g.deck.Shuffle(seed)
	}

	for _, p := range g.idToParticipant {
		p.newRound()
	}

	for _, p := range g.participants {
		card, err := g.deck.Draw()
		if err != nil {
			return err
		}

		p.card = card
	}

	g.dealtCards = true
	return nil
}

// flipAllCards must only be called at the end of the game, or after the dealer announced they are going to the
// deck. Validation is assumed to happen elsewhere
func (g *Game) flipAllCards() {
	for _, p := range g.participants {
		p.isFlipped = true
	}
}

func (g *Game) isGameOver() bool {
	return g.balanceAdjustments != nil
}

func (g *Game) isRoundOver() bool {
	return g.loserGroups != nil
}

// -- Methods for the playable.Playable interface --

// Name returns the name of the game
func (g *Game) Name() string {
	return fmt.Sprintf("Pass the Poop, %s Edition", g.options.Edition.Name())
}

// Action is called when a client performs an action
// Part of the Playable interface
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	switch message.Action {
	case "execute":
		raw, err := strconv.Atoi(message.Subject)
		if err != nil {
			return nil, false, err
		}

		action, err := GameActionFromInt(raw)
		if err != nil {
			return nil, false, err
		}

		switch action {
		case ActionEndRound:
			if err := g.EndRound(); err != nil {
				return nil, false, err
			}

			g.nextRoundStartTime = time.Now().Add(nextRoundDelay)

			return playable.OK(), true, nil
		case ActionNextRound:
			diff := g.nextRoundStartTime.Sub(time.Now())
			if !g.nextRoundStartTime.IsZero() && diff > 0 {
				return nil, false, fmt.Errorf("please wait %.1f s until starting the next round", float64(diff) / float64(time.Second))
			}

			if err := g.nextRound(); err != nil {
				return nil, false, err
			}

			return playable.OK(), true, nil
		case ActionEndGame:
			g.endGameAck = true
			return playable.OK(), true, nil
		default:
			if err := g.ExecuteTurnForPlayer(playerID, action); err != nil {
				return nil, false, err
			}

			return playable.OK(), true, nil
		}
	default:
		return nil, false, fmt.Errorf("unsupported action: %s", message.Action)
	}
}

// GetPlayerState returns the player state in the game
// Part of the Playable interface
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	var card *deck.Card
	actions := make([]GameAction, 0)
	currentTurn := int64(0)
	if p := g.getCurrentTurn(); p != nil {
		currentTurn = p.PlayerID
	}

	participant, found := g.idToParticipant[playerID]
	if found {
		card = participant.card

		if currentTurn == playerID {
			if g.pendingTrade {
				if participant.card.Rank == deck.King {
					actions = []GameAction{ActionFlipKing}
				} else {
					actions = []GameAction{ActionAccept}
				}
			} else {
				if g.isDealersTurn() {
					if g.dealerWillGoToDeck {
						actions = []GameAction{ActionDrawFromDeck}
					} else if participant.card.Rank == deck.King {
						actions = []GameAction{ActionStay}
					} else {
						actions = []GameAction{ActionStay, ActionGoToDeck}
					}
				} else {
					actions = []GameAction{ActionStay, ActionTrade}
				}
			}
		}

		if g.isGameOver() {
			actions = append(actions, ActionEndGame)
		} else if g.isRoundOver() {
			actions = append(actions, ActionNextRound)
		} else if g.getCurrentTurn() == nil {
			actions = append(actions, ActionEndRound)
		}
	}

	return &playable.Response{
		Key:   "game",
		Value: "pass-the-poop",
		Data: &ParticipantState{
			Participant:      participant,
			Card:             card,
			AvailableActions: actions,
			GameState: &GameState{
				Edition:         g.options.Edition.Name(),
				Participants:    g.participants,
				AllParticipants: g.idToParticipant,
				Ante:            g.options.Ante,
				Pot:             g.pot,
				CurrentTurn:     currentTurn,
				LastGameAction:  g.lastGameAction,
				LoserGroups:     g.loserGroups,
			},
		},
	}, nil
}

// GetEndOfGameDetails returns the final results
// Part of the Playable interface
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	if !g.endGameAck {
		return nil, false
	}

	return &playable.GameOverDetails{
		BalanceAdjustments: g.balanceAdjustments,
		Log:                nil,
	}, true
}

// LogChan returns a channel where log messages will be sent
// Part of the Playable interface
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}

func (g *Game) sendLogMessage(playerID int64, msg string, card ...*deck.Card) {
	g.logChan <- []*playable.LogMessage{newLogMessage(playerID, msg, card...)}
}

func newLogMessage(playerID int64, msg string, card ...*deck.Card) *playable.LogMessage {
	return &playable.LogMessage{
		UUID:      uuid.New().String(),
		PlayerIDs: []int64{playerID},
		Cards:     card,
		Message:   msg,
		Time:      time.Now(),
	}
}
