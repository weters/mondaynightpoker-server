package playable

// Player is a player in a playable game
type Player interface {
	GetPlayerID() int64
	GetTableStake() int
}
