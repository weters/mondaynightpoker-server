package playable

import "time"

// Tickable is an interface that allows a periodic tick to update the game state
type Tickable interface {
	// Interval is how long the wait between each tick should be
	Interval() time.Duration

	// Tick will be called periodically
	// Return true if the dealer should request updated data
	Tick() (bool, error)
}
