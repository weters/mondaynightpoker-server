package passthepoop

import (
	"math"
	"mondaynightpoker-server/pkg/deck"
)

// PairsEdition is a variant of Pass the Poop where pairs on the board good
// Any pair on the board is better than any single card
// Trips or better on the board and the rest of the board loses all their lives
type PairsEdition struct {
}

// Name returns the name of the Edition
func (p *PairsEdition) Name() string {
	return "Pairs"
}

// ParticipantWasPassed is a no-op in pairs edition
func (p *PairsEdition) ParticipantWasPassed(participant *Participant, nextCard *deck.Card) {
	// noop
}

// EndRound ends the round
// In pairs edition, and match on the board beats any single high card. For example,
// If two people turn over aces, those "pair" of aces beat someone with a King.
// In the event three or four people all have the same card, the rest of the table loses their entire stack
// of lives
func (p *PairsEdition) EndRound(participants []*Participant) ([]*LoserGroup, error) {
	cardStats := make(map[int][]*Participant)
	largestGroupSize := 0
	largestGroupRank := -1

	for _, participant := range participants {
		rank := participant.card.AceLowRank()

		group, found := cardStats[rank]
		if !found {
			group = []*Participant{participant}
		} else {
			group = append(group, participant)
		}

		nInGroup := len(group)
		if nInGroup == largestGroupSize {
			if rank > largestGroupRank {
				largestGroupRank = rank
			}
		} else if nInGroup > largestGroupSize {
			largestGroupSize = nInGroup
			largestGroupRank = rank
		}

		cardStats[rank] = group
	}

	// trips or better and the rest lose
	if largestGroupSize >= 3 {
		roundLosers := make([]*RoundLoser, 0, len(participants)-largestGroupSize)
		for _, participant := range participants {
			if participant.card.AceLowRank() != largestGroupRank {
				roundLosers = append(roundLosers, &RoundLoser{
					PlayerID:  participant.PlayerID,
					Card:      participant.card,
					LivesLost: participant.subtractLife(0),
				})
			}
		}

		return newLoserGroup(roundLosers), nil
	}

	// otherwise, find the lowest rank in the smallest group
	lowestRank := math.MaxInt32
	smallestGroupSize := math.MaxInt32
	for rank, participants := range cardStats {
		nParticipants := len(participants)
		if nParticipants > smallestGroupSize {
			continue
		} else if nParticipants == smallestGroupSize {
			if rank < lowestRank {
				lowestRank = rank
			}
		} else {
			smallestGroupSize = nParticipants
			lowestRank = rank
		}
	}

	if lowestRank == math.MaxInt32 {
		// this should never happen
		panic("could not find lowest card")
	}

	losingParticipants := cardStats[lowestRank]
	roundLosers := make([]*RoundLoser, len(losingParticipants))

	if len(participants) == len(losingParticipants) {
		return nil, ErrMutualDestruction
	}

	for i, participant := range losingParticipants {
		roundLosers[i] = &RoundLoser{
			PlayerID:  participant.PlayerID,
			Card:      participant.card,
			LivesLost: participant.subtractLife(1),
		}
	}

	return newLoserGroup(roundLosers), nil
}
