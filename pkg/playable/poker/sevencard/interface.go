package sevencard

// Variant is a specific variant of seven-card poker (i.e., Stud, Baseball, Chicago, etc.)
type Variant interface {
	// Name should return the name of the game
	Name() string
}
