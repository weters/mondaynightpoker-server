package guts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, 25, opts.Ante)
	assert.Equal(t, 1000, opts.MaxOwed)
}
