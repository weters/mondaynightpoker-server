package littlel

// Options provides options for the Little L game
type Options struct {
	Ante int
	// InitialDeal is how many cards each player is initially dealt
	InitialDeal int
	// TradeIns is how many cards the player may trade-in
	TradeIns []int
}

// DefaultOptions returns the default set of options
func DefaultOptions() Options {
	return Options{
		Ante:        25,
		InitialDeal: 4,
		TradeIns:    []int{0, 2},
	}
}
