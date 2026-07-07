package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func resetSpeed(t *testing.T) {
	t.Helper()
	prev := autoPilotSpeed.Load()
	t.Cleanup(func() { autoPilotSpeed.Store(prev) })
	autoPilotSpeed.Store(int32(speedNormal))
}

func TestCycleSpeed(t *testing.T) {
	resetSpeed(t)

	assert.Equal(t, speedNormal, currentSpeed())
	assert.Equal(t, speedFast, cycleSpeed())
	assert.Equal(t, speedInstant, cycleSpeed())
	assert.Equal(t, speedSlow, cycleSpeed())
	assert.Equal(t, speedNormal, cycleSpeed())
}

func TestSetSpeed(t *testing.T) {
	resetSpeed(t)

	assert.True(t, setSpeed("instant"))
	assert.Equal(t, speedInstant, currentSpeed())
	assert.True(t, setSpeed("fast"))
	assert.Equal(t, speedFast, currentSpeed())
	assert.True(t, setSpeed("slow"))
	assert.Equal(t, speedSlow, currentSpeed())
	assert.True(t, setSpeed("normal"))
	assert.Equal(t, speedNormal, currentSpeed())

	assert.False(t, setSpeed("warp"))
	assert.Equal(t, speedNormal, currentSpeed())
}

func TestSpeedString(t *testing.T) {
	assert.Equal(t, "normal", speedNormal.String())
	assert.Equal(t, "fast", speedFast.String())
	assert.Equal(t, "instant", speedInstant.String())
	assert.Equal(t, "slow", speedSlow.String())
}

func TestSpeedDelay(t *testing.T) {
	assert.Equal(t, time.Duration(0), speedInstant.delay())

	for range 20 {
		d := speedFast.delay()
		assert.GreaterOrEqual(t, d, 50*time.Millisecond)
		assert.Less(t, d, 250*time.Millisecond)

		d = speedNormal.delay()
		assert.GreaterOrEqual(t, d, 500*time.Millisecond)
		assert.Less(t, d, 2000*time.Millisecond)

		d = speedSlow.delay()
		assert.GreaterOrEqual(t, d, 1500*time.Millisecond)
		assert.Less(t, d, 3000*time.Millisecond)
	}
}
