package passthepoop

import (
	"errors"
	"math"
	"mondaynightpoker-server/pkg/deck"
)

// ErrMutualDestruction is an error when all the players who lost have one life left and nobody
// would otherwise be left
var ErrMutualDestruction = errors.New("game cannot end in a tie")

// StandardEdition is the standard variant of Pass the Poop
type StandardEdition struct {
}

// EndRound ends the round
func (s *StandardEdition) EndRound(participants []*Participant) ([]*LoserGroup, error) {
	lowRank := math.MaxInt32 // non-existent rank
	low := make([]*Participant, 0, 1)
	for _, participant := range participants {
		// note: dead cards don't apply in standard edition
		pRank := participant.card.Rank
		if pRank == deck.Ace {
			// ace is the lowest card
			pRank = 0
		}

		if pRank == lowRank {
			low = append(low, participant)
		} else if pRank < lowRank {
			low = []*Participant{participant}
			lowRank = pRank
		} // else, higher card, thus safe
	}

	if len(low) == 0 {
		// this should never happen
		return nil, errors.New("no loser determined")
	}

	// there's at least one winner left
	if len(low) == len(participants) {
		// loser group and all participants overlap
		atLeastOneSurvivor := false
		for _, p := range low {
			if p.lives > 1 {
				atLeastOneSurvivor = true
				break
			}
		}

		if !atLeastOneSurvivor {
			return nil, ErrMutualDestruction
		}
	}

	roundLosers := make([]*RoundLoser, len(low))
	for i, p := range low {
		livesLost := p.subtractLife(1)
		roundLosers[i] = &RoundLoser{
			PlayerID:  p.PlayerID,
			Card:      p.card,
			LivesLost: livesLost,
		}
	}

	return []*LoserGroup{
		{
			Order:       0,
			RoundLosers: roundLosers,
		},
	}, nil
}

// Name returns the name of the Edition
func (s *StandardEdition) Name() string {
	return "Standard"
}

// ParticipantWasPassed is a no-op in standard edition
func (s *StandardEdition) ParticipantWasPassed(participant *Participant, nextCard *deck.Card) {
	// noop
}
