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
	deck            *deck.Deck
	participants    []*Participant
	idToParticipant map[int64]*Participant

	decisionIndex int
	// prevent deal() from being called multiple times
	dealtCards bool
}

// GameAction is a game action a player can take (i.e., stay or trade)
type GameAction int

// game action constants
const (
	ActionStay GameAction = iota
	ActionTrade
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
	for i, id := range playerIDs {
		participants[i] = &Participant{
			PlayerID: id,
			lives:    options.Lives,
			balance:  -1 * options.Ante,
		}
		idToParticipants[id] = participants[i]
	}

	g := &Game{
		deck:            d,
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

// PerformGameAction will perform a game action for the player
// A player can either stay or trade
// If trading, the player can either trade with the next player, or if they are the last player, they can go to the deck
// If trading with a player, and the next player has a King, they cannot trade
func (g *Game) PerformGameAction(playerID int64, gameAction GameAction) (ActionResult, error) {
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
	case ActionTrade:
		if participant.card.Rank == deck.King {
			return ResultError, errors.New("you cannot trade a king")
		}

		if g.decisionIndex+1 == len(g.participants) {
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

// decisionWith returns the participant who is currently making the decision
func (g *Game) decisionWith() *Participant {
	return g.participants[g.decisionIndex]
}

// nextRound will handle cleanup and set state for next round
// 1. Determine next dealer
// 2. Remove dead players
// 3. Set next decision index
func (g *Game) nextRound() error {
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

	if len(newList) <= 1 {
		return errors.New("expected to find at least two active players left")
	}

	g.participants = newList
	g.dealtCards = false
	g.decisionIndex = 0
	return nil
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

// canReveal returns true if all players have had a turn
func (g *Game) canReveal() bool {
	return g.decisionIndex == len(g.participants)
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
	panic("implement me")
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
