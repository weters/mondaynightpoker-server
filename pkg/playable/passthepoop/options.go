package passthepoop

// Options provides options for the game
type Options struct {
	// Ante is the total ante for the game
	Ante int
	// Lives are how many rounds a player can lose
	Lives int
	// Edition is the game variant
	Edition Edition
}

// DefaultOptions returns the default options
func DefaultOptions() Options {
	return Options{
		Ante:    75,
		Lives:   3,
		Edition: &StandardEdition{},
	}
}
