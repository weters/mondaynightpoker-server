package util

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestGetRandomName(t *testing.T) {
	rand.Seed(0)
	assert.Equal(t, "Waiving Lion", GetRandomName())
	assert.Equal(t, "Jumping Bear", GetRandomName())
}
