package passthepoop

import (
	"math"
	"mondaynightpoker-server/pkg/deck"
)

// DiarrheaEdition is a faster variant of Pass the Poop with the following stipulations
// 1. An ace passed back results in the loss of a life, the ace is thrown out, and we still play for next low card
// 2. If multiple people are tied with the lowest card, they lose all their lives, the cards are thrown out, and we still play for next low card
type DiarrheaEdition struct {
}

// Name returns the name of the Edition
func (d *DiarrheaEdition) Name() string {
	return "Diarrhea"
}

// ParticipantWasPassed will check if an ace was passed back
func (d *DiarrheaEdition) ParticipantWasPassed(participant *Participant, nextCard *deck.Card) {
	if nextCard.Rank == deck.Ace {
		participant.deadCard = true
		participant.subtractLife(1)
	}
}

// EndRound ends the round
func (d *DiarrheaEdition) EndRound(participants []*Participant) ([]*LoserGroup, error) {
	if len(participants) <= 1 {
		panic("EndRound should not be called with a single participant")
	}

	aceLosers := make([]*RoundLoser, 0)

	// filter out any participants who have a dead card (i.e., were passed back an Ace)
	liveParticipants := make([]*Participant, 0, len(participants))
	for _, participant := range participants {
		if !participant.deadCard {
			liveParticipants = append(liveParticipants, participant)
		} else {
			aceLosers = append(aceLosers, &RoundLoser{
				PlayerID:  participant.PlayerID,
				Card:      participant.card,
				LivesLost: 1, // XXX is this the best way to do this???
			})
		}
	}

	loserGroups := make([]*LoserGroup, 0)
	if len(aceLosers) > 0 {
		loserGroups = append(loserGroups, &LoserGroup{
			Order:       0,
			RoundLosers: aceLosers,
		})
	}

	if len(liveParticipants) <= 1 {
		if len(loserGroups) == 0 {
			panic("can't have 1 live participant and no loser groups")
		}

		if len(liveParticipants) == 1 {
			return loserGroups, nil
		}

		// no live participants left. see if the one's who got an ace at least
		// one has a life left
		roundOK := false
		for _, p := range participants {
			if p.deadCard && p.lives > 0 {
				roundOK = true
				break
			}
		}

		// nobody has any lives left. add a life
		if !roundOK {
			for _, p := range participants {
				if p.deadCard {
					p.lives++
				}
			}

			return nil, ErrMutualDestruction
		}

		return loserGroups, nil
	}

	for i := len(loserGroups); true; i++ {
		if i > 52 {
			panic("infite loop detected")
		}

		roundLosers, remainingParticipants := d.getLowCards(liveParticipants)
		if roundLosers == nil {
			if i == 0 {
				return nil, ErrMutualDestruction
			}

			// keep last remaining players
			return loserGroups, nil
		}

		loserGroups = append(loserGroups, &LoserGroup{
			Order:       i,
			RoundLosers: roundLosers,
		})

		// only one loser (i.e., no automatic loss) or one participant remaining (i.e. winner)
		// then return
		if len(roundLosers) == 1 || len(remainingParticipants) == 1 {
			return loserGroups, nil
		}

		liveParticipants = remainingParticipants
	}

	panic("this should not be hit")
}

// nil, nil is returned if there is a double screwed and no remaining players
func (d *DiarrheaEdition) getLowCards(participants []*Participant) ([]*RoundLoser, []*Participant) {
	lowRank := math.MaxInt32
	low := make(map[*Participant]int)
	index := 0
	for _, participant := range participants {
		rank := participant.card.Rank
		if rank == deck.Ace {
			rank = 0
		}

		if rank < lowRank {
			lowRank = rank
			index = 0
			low = map[*Participant]int{
				participant: 0,
			}
		} else if rank == lowRank {
			index++
			low[participant] = index
		}
	}

	remainingParticipants := make([]*Participant, 0, len(participants))
	for _, p := range participants {
		if _, isLow := low[p]; !isLow {
			remainingParticipants = append(remainingParticipants, p)
		}
	}

	// cannot double-out with no remaining players
	if len(low) > 1 && len(remainingParticipants) == 0 {
		return nil, nil
	}

	subLife := 1
	if len(low) > 1 {
		subLife = 0 // all lives lost
	}
	roundLosers := make([]*RoundLoser, len(low))
	for p, index := range low {
		livesLost := p.subtractLife(subLife)
		roundLosers[index] = &RoundLoser{
			PlayerID:  p.PlayerID,
			Card:      p.card,
			LivesLost: livesLost,
		}
	}

	return roundLosers, remainingParticipants
}
