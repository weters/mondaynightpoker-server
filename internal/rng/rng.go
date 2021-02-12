package rng

// Generator provides a simple random number
type Generator interface {
	// Intn will return a random number up to but not including n
	Intn(n int) int
}
