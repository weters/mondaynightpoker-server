package passthepoop

// Options provides options for the game
type Options struct {
	// Ante is the total ante for the game
	Ante int
	// Lives are how many rounds a player can lose
	Lives int
	// Edition is the game variant
	Edition Edition
	// AllowBlocks will give the player one block to use
	AllowBlocks bool
	// Seed is a deterministic shuffle seed for tests; -1 (the default) uses
	// a crypto-secure shuffle. Never populated from client input.
	Seed int64
}

// DefaultOptions returns the default options
func DefaultOptions() Options {
	return Options{
		Ante:        75,
		Lives:       3,
		Edition:     &StandardEdition{},
		AllowBlocks: false,
		Seed:        -1,
	}
}
