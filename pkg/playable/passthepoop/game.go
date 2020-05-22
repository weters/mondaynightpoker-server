package passthepoop

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
)

// Game is an individual game of pass the poop
type Game struct {
	options         Options
	pot             int
	deck            *deck.Deck
	participants    []*Participant
	idToParticipant map[int64]*Participant

	decisionIndex int
	// did the dealer decide to go to the deck?
	dealerWillGoToDeck bool
	// prevent deal() from being called multiple times
	dealtCards bool
}

// GameAction is a game action a player can take (i.e., stay or trade)
type GameAction int

// game action constants
const (
	ActionStay GameAction = iota
	ActionTrade
	// ActionGoToDeck happens when the dealer announces their intention to go to the deck
	// They will have to follow it up with an ActionTrade as well to actually execute that move
	ActionGoToDeck
)

// ActionResult is the result of a player's action
type ActionResult int

// action result values
const (
	// ResultError is no result
	ResultError ActionResult = iota
	// ResultOK means the trade or stay was successful
	ResultOK
	// ResultKing means the move was blocked by a King
	ResultKing
)

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
	}

	if err := g.deal(); err != nil {
		return nil, err
	}

	return g, nil
}

// ExecuteTurnForPlayer will perform a game action for the player
// A player can either stay or trade
// If trading, the player can either trade with the next player, or if they are the last player, they can go to the deck
// If trading with a player, and the next player has a King, they cannot trade
func (g *Game) ExecuteTurnForPlayer(playerID int64, gameAction GameAction) (ActionResult, error) {
	if g.decisionIndex >= len(g.participants) {
		return ResultError, errors.New("no more decisions can be made this round")
	}

	participant, ok := g.idToParticipant[playerID]
	if !ok {
		return ResultError, fmt.Errorf("%d is not in this game", playerID)
	}

	if participant != g.decisionWith() {
		return ResultError, errors.New("you are not up")
	}

	switch gameAction {
	case ActionStay:
		// do nothing
	case ActionGoToDeck:
		if !g.isDecisionWithDealer() {
			return ResultError, errors.New("only the dealer may go to the deck")
		}

		// going to the deck is a two-step process so we can first reveal the rest of the cards so the players
		// can see what the dealer needs. #Drama
		g.dealerWillGoToDeck = true

		g.flipAllCards()

		// return early so we don't move the decision index
		return ResultOK, nil
	case ActionTrade:
		if participant.card.Rank == deck.King {
			return ResultError, errors.New("you cannot trade a king")
		}

		if g.isDecisionWithDealer() {
			if !g.dealerWillGoToDeck {
				return ResultError, errors.New("you must first announce your intention to go to the deck")
			}

			// go to the deck
			nextCard, err := g.deck.Draw()
			if err != nil {
				return ResultError, err
			}

			participant.card = nextCard
		} else {
			// swap with the next player
			nextParticipant := g.participants[g.decisionIndex+1]
			nextCard := nextParticipant.card
			if nextCard.Rank == deck.King {
				// cannot trade into a king
				g.decisionIndex++
				return ResultKing, nil
			}

			// players swap cards
			participant.card, nextParticipant.card = nextCard, participant.card
		}
	default:
		return ResultError, fmt.Errorf("not a valid game action")
	}

	g.decisionIndex++
	return ResultOK, nil
}

// EndRound performs all necessary end of round actions
func (g *Game) EndRound() error {
	g.flipAllCards()

	loserGroups, err := g.options.Edition.EndRound(g.participants)
	if err != nil {
		return err
	}

	// TODO: do something with the loser groups
	_ = loserGroups

	return nil
}

// decisionWith returns the participant who is currently making the decision
func (g *Game) decisionWith() *Participant {
	return g.participants[g.decisionIndex]
}

// isDecisionWithDealer returns true if the dealer is up
func (g *Game) isDecisionWithDealer() bool {
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

	g.dealtCards = false
	g.decisionIndex = 0
	g.dealerWillGoToDeck = false

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

	for _, p := range g.participants {
		card, err := g.deck.Draw()
		if err != nil {
			return err
		}

		p.newRound()
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

// -- Methods for the playable.Playable interface --

// Name returns the name of the game
func (g *Game) Name() string {
	return fmt.Sprintf("Pass the Poop, %s Edition", g.options.Edition.Name())
}

// Action is called when a client performs an action
// Part of the Playable interface
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	panic("implement me")
}

// GetPlayerState returns the player state in the game
// Part of the Playable interface
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	participant, found := g.idToParticipant[playerID]
	if !found {
		return nil, fmt.Errorf("could not find player with ID %d", playerID)
	}

	return &playable.Response{
		Key:   "game",
		Value: "pass-the-poop",
		Data: &ParticipantState{
			Participant: participant,
			Card:        participant.card,
			GameState: &GameState{
				Edition:         g.options.Edition.Name(),
				Participants:    g.participants,
				AllParticipants: g.idToParticipant,
				Ante:            g.options.Ante,
				Pot:             g.pot,
				DecisionIndex:   g.decisionIndex,
			},
		},
	}, nil
}

// GetEndOfGameDetails returns the final results
// Part of the Playable interface
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	panic("implement me")
}

// LogChan returns a channel where log messages will be sent
// Part of the Playable interface
func (g *Game) LogChan() chan []*playable.LogMessage {
	panic("implement me")
}
