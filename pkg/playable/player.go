package playable

// Player is a player in a playable game
type Player interface {
	GetPlayerID() int64
	GetTableStake() int
}

// SimplePlayer is a minimal Player implementation with no table stake.
// It backs the deprecated NewGame constructors and tests that only have player IDs.
type SimplePlayer struct {
	ID         int64
	TableStake int
}

// GetPlayerID returns the player ID
func (s *SimplePlayer) GetPlayerID() int64 {
	return s.ID
}

// GetTableStake returns the player's table stake
func (s *SimplePlayer) GetTableStake() int {
	return s.TableStake
}

// SimplePlayers converts a list of player IDs into Player implementations with no table stake
func SimplePlayers(ids []int64) []Player {
	players := make([]Player, len(ids))
	for i, id := range ids {
		players[i] = &SimplePlayer{ID: id}
	}

	return players
}

// PlayerIDs extracts the player IDs from a list of players
func PlayerIDs(players []Player) []int64 {
	ids := make([]int64, len(players))
	for i, player := range players {
		ids[i] = player.GetPlayerID()
	}

	return ids
}
