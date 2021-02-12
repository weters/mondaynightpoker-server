package rng

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCrypto_Intn(t *testing.T) {
	a := assert.New(t)

	c := Crypto{}
	found := make(map[int]bool)
	// it's possible this could fail, but not likely
	for i := 0; i < 1000; i++ {
		found[c.Intn(5)] = true
	}

	a.True(found[0])
	a.True(found[1])
	a.True(found[2])
	a.True(found[3])
	a.True(found[4])
	a.False(found[5])
}
