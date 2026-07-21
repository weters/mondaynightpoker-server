package money

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatCents(t *testing.T) {
	tests := []struct {
		name     string
		cents    int
		expected string
	}{
		{"zero", 0, "$0"},
		{"whole dollar", 200, "$2"},
		{"dollars and cents", 150, "$1.50"},
		{"sub-dollar pads to two places", 5, "$0.05"},
		{"sub-dollar no padding needed", 99, "$0.99"},
		{"large amount", 1234567, "$12345.67"},
		{"negative whole dollar", -200, "-$2"},
		{"negative dollars and cents", -150, "-$1.50"},
		{"negative sub-dollar", -50, "-$0.50"},
		{"negative pads to two places", -5, "-$0.05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatCents(tt.cents))
		})
	}
}

// TestFormatCents_minInt guards the negation overflow: -math.MinInt is still
// negative, which would otherwise emit a string with an embedded minus sign.
func TestFormatCents_minInt(t *testing.T) {
	got := FormatCents(math.MinInt)
	assert.Equal(t, "-", got[:1])
	assert.NotContains(t, got[1:], "-")
}
