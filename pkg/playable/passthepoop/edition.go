package passthepoop

import "mondaynightpoker-server/pkg/deck"

// RoundLoser provides details for a particular participant who lost a round
type RoundLoser struct {
	PlayerID  int64      `json:"playerId"`
	Card      *deck.Card `json:"card"`
	LivesLost int        `json:"livesLost"`
}

// LoserGroup is a group of losers who lost together
type LoserGroup struct {
	Order       int           `json:"order"`
	RoundLosers []*RoundLoser `json:"roundLosers"`
	// Reason is an optional human-readable description of why this group lost
	// (e.g. "trips of Aces", "mutual low card"). Used for game logs.
	Reason string `json:"reason,omitempty"`
}

func newLoserGroup(roundLosers ...[]*RoundLoser) []*LoserGroup {
	lg := make([]*LoserGroup, len(roundLosers))
	for i, rl := range roundLosers {
		lg[i] = &LoserGroup{
			Order:       i,
			RoundLosers: rl,
		}
	}

	return lg
}

// Edition provides capabilities for a specific variant of Pass the Poop
type Edition interface {
	// Name returns the name of the Edition
	Name() string
	// ParticipantWasPassed  performs any actions on a pass back
	ParticipantWasPassed(participant *Participant, nextCard *deck.Card)

	// EndRound performs all end of round calculations
	EndRound(participants []*Participant) ([]*LoserGroup, error)
}
