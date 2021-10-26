package util

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestGetRandomName(t *testing.T) {
	random = rand.New(rand.NewSource(0)) // nolint:gosec
	assert.Equal(t, "Waiving Lion", GetRandomName())
	assert.Equal(t, "Jumping Bear", GetRandomName())
}
