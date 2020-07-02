package poker

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHand_String(t *testing.T) {
	assert.PanicsWithValue(t, "unknown hand: -1", func() {
		_ = Hand(-1).String()
	})
}
