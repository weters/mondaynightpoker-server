package sevencard

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/deck"
)

const maxParticipants = 7

var seed int64 = 0

// Game is a single game of seven-card poker
type Game struct {
	deck            *deck.Deck
	round           round
	options         Options
	playerIDs       []int64
	idToParticipant map[int64]*participant
}

// NewGame returns a new seven-card poker Game
func NewGame(tableUUID string, playerIDs []int64, options Options) (*Game, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	if len(playerIDs) < 2 {
		return nil, errors.New("you must have at least two participants")
	}

	if len(playerIDs) > maxParticipants {
		return nil, fmt.Errorf("seven-card allows at most %d participants", maxParticipants)
	}

	d := deck.New()
	d.Shuffle(seed)

	return &Game{
		deck:            d,
		options:         options,
		playerIDs:       append([]int64{}, playerIDs...), // copy
		idToParticipant: buildIDToParticipant(playerIDs),
	}, nil
}

func (g *Game) Start() error {
	if g.round != beforeDeal {
		return errors.New("the game has already started")
	}

	// deal two face-down, one face-up
	for _, faceDown := range []bool{true, true, false} {
		if err := g.dealCards(faceDown); err != nil {
			return err
		}
	}

	g.round++
	return nil
}

func (g *Game) dealCards(faceDown bool) error {
	for _, pid := range g.playerIDs {
		player := g.idToParticipant[pid]
		if !player.didFold {
			card, err := g.deck.Draw()
			if err != nil {
				return err
			}

			if !faceDown {
				card.BitField |= faceUp
			}

			player.hand.AddCard(card)
		}
	}

	return nil
}

func buildIDToParticipant(playerIDs []int64) map[int64]*participant {
	i2p := make(map[int64]*participant)
	for _, pid := range playerIDs {
		i2p[pid] = newParticipant(pid)
	}

	return i2p
}
