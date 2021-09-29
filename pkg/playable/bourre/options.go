package bourre

// Options are options for creating a new bourre game
type Options struct {
	InitialPot int
	Ante       int
	FiveSuit   bool
}

// DefaultOptions returns the default options
func DefaultOptions() Options {
	return Options{
		InitialPot: 0,
		Ante:       50,
		FiveSuit:   false,
	}
}
