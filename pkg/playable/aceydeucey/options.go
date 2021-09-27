package aceydeucey

// Options contains options for creating a new game of Acey Deucey
type Options struct {
	Ante      int
	AllowPass bool
	GameType  GameType
}

// DefaultOptions returns the default set of options
func DefaultOptions() Options {
	return Options{
		Ante:      25,
		AllowPass: false,
		GameType:  GameTypeStandard,
	}
}
