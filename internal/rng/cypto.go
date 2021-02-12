package rng

import (
	"crypto/rand"
	"math/big"
)

// Crypto wraps the crypto/rand library
type Crypto struct{}

// Intn returns a random number from 0 < n
func (c Crypto) Intn(n int) int {
	b, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}

	return int(b.Int64())
}
