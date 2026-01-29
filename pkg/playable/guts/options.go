package guts

// Options are options for creating a new guts game
type Options struct {
	Ante      int // Default: 25 cents
	MaxOwed   int // Default: 1000 ($10), capped penalty amount
	CardCount int // 2 or 3, defaults to 2
}

// DefaultOptions returns the default options for a guts game
func DefaultOptions() Options {
	return Options{
		Ante:      25,
		MaxOwed:   1000, // $10.00
		CardCount: 2,
	}
}
