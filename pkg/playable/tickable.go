package playable

import "time"

type Tickable interface {
	// Delay is how long the wait between each tick should be
	Delay() time.Duration

	// Tick will be called periodically
	// Return true if the dealer should request updated data
	Tick() (bool, error)
}
