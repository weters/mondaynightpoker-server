package sevencard

// Game is a single game of seven-card poker
type Game struct {
}

// NewGame returns a new seven-card poker Game
func NewGame(tableUUID string, playerIDs []int64, options Options) (*Game, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	return &Game{}, nil
}
