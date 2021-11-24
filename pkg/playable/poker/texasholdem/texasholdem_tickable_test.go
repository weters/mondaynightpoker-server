package texasholdem

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGame_Interval(t *testing.T) {
	game := setupNewGame(DefaultOptions(), 50, 50)
	assert.Equal(t, time.Second, game.Interval())
}
