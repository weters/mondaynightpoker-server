package passthepoop

import (
	"mondaynightpoker-server/pkg/deck"
	"time"
)

// GameLog keeps track of the game log
type GameLog struct {
	Edition   string          `json:"edition"`
	Pot       int             `json:"pot"`
	Ante      int             `json:"ante"`
	Lives     int             `json:"lives"`
	Rounds    []*GameLogRound `json:"rounds"`
	StartTime time.Time       `json:"startTime"`
	Players   []int64         `json:"players"`
	Winner    int64           `json:"winner"`
}

// GameLogHand represent's an individual player's hand
type GameLogHand struct {
	PlayerID int64      `json:"playerId"`
	Card     *deck.Card `json:"card"`
}

// AddRound adds a new round
func (g *GameLog) AddRound(startingHand []*GameLogHand) {
	glr := &GameLogRound{
		Round:        len(g.Rounds),
		StartingHand: startingHand,
		GameActions:  make([]*GameActionDetails, 0),
	}

	g.Rounds = append(g.Rounds, glr)
}

func (g *GameLog) lastRound() *GameLogRound {
	last := len(g.Rounds) - 1
	return g.Rounds[last]
}

// EndRound sets the end time
func (g *GameLog) EndRound() {
	g.lastRound().EndTime = time.Now()
}

// AddGameAction adds a game action
func (g *GameLog) AddGameAction(gad *GameActionDetails) {
	lr := g.lastRound()
	lr.GameActions = append(lr.GameActions, gad)
}

// SetLoserGroups will set the loser groups
func (g *GameLog) SetLoserGroups(loserGroups []*LoserGroup) {
	g.lastRound().LoserGroups = loserGroups
}

// GameLogRound is an individual round
type GameLogRound struct {
	Round        int                  `json:"round"`
	EndTime      time.Time            `json:"endTime"`
	StartingHand []*GameLogHand       `json:"startingHand"`
	GameActions  []*GameActionDetails `json:"gameActions"`
	LoserGroups  []*LoserGroup        `json:"loserGroups"`
}
