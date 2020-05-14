package bourre

import (
	"errors"
)

// Result contain the results from a completed game of bourré
type Result struct {
	Parent *Result

	PaidAnte      []*Player
	PaidPot       []*Player
	Winners       []*Player
	Folded        []*Player
	Booted        []*Player
	WinningAmount int
	Ante          int
	OldPot        int
	NewPot        int

	table       string
	playerOrder map[*Player]int
	idToPlayer  map[int64]*Player
}

// ShouldContinue checks whether another game can be created
func (r *Result) ShouldContinue() bool {
	return r.NewPot > 0
}

// ErrCannotCreateGame happens when you try to create a new game from a result when there's no pot
var ErrCannotCreateGame = errors.New("cannot create a new game from an existing game")

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

	g, err := newGame(nextPlayers, r.Folded, Options{InitialPot: r.NewPot, Ante: r.Ante})
	if err != nil {
		return nil, err
	}

	g.idToPlayer = r.idToPlayer
	g.parentResult = r
	g.table = r.table

	return g, nil
}
