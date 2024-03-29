package bourre

import (
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/playable"
)

// Result contain the results from a completed game of bourré
type Result struct {
	Parent *Result

	Options       Options
	PaidAnte      []*Player
	PaidPot       []*Player
	Winners       []*Player
	Folded        []*Player
	Booted        []*Player
	WinningAmount int
	Ante          int
	OldPot        int
	NewPot        int

	logger      logrus.FieldLogger
	logChan     chan []*playable.LogMessage
	playerOrder map[*Player]int
	idToPlayer  map[int64]*Player
}

// ShouldContinue checks whether another game can be created
func (r *Result) ShouldContinue() bool {
	return r.NewPot > 0
}

// NewGame can be called if a bourré round ends and there's still a live pot
func (r *Result) NewGame() (*Game, error) {
	if !r.ShouldContinue() {
		return nil, ErrCannotCreateGame
	}

	players := make([]*Player, len(r.playerOrder))
	for player, i := range r.playerOrder {
		players[i] = player
	}

	first := players[0]
	copy(players, players[1:])
	players[len(players)-1] = first

	wasBooted := make(map[*Player]bool)
	for _, player := range r.Booted {
		wasBooted[player] = true
	}

	nextPlayers := make([]*Player, 0, len(players))
	for _, player := range players {
		if _, booted := wasBooted[player]; booted {
			continue
		}

		if player.folded {
			continue
		}

		player.NewGame()
		nextPlayers = append(nextPlayers, player)
	}

	folded := append(r.Folded, r.Booted...)

	opts := r.Options
	opts.InitialPot = r.NewPot
	opts.Ante = r.Ante

	g, err := newGame(r.logger, nextPlayers, folded, opts)
	if err != nil {
		return nil, err
	}

	g.idToPlayer = r.idToPlayer
	g.parentResult = r
	g.logger = r.logger
	g.logChan = r.logChan

	return g, nil
}
