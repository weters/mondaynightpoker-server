package main

import (
	"sync/atomic"
	"time"
)

// speedLevel controls how long auto-pilot bots "think" before acting.
type speedLevel int32

const (
	speedNormal speedLevel = iota
	speedFast
	speedInstant
	speedSlow
	numSpeedLevels
)

// autoPilotSpeed is the process-wide auto-pilot speed setting.
var autoPilotSpeed atomic.Int32

func currentSpeed() speedLevel {
	return speedLevel(autoPilotSpeed.Load())
}

// cycleSpeed advances to the next speed level and returns it.
func cycleSpeed() speedLevel {
	next := (currentSpeed() + 1) % numSpeedLevels
	autoPilotSpeed.Store(int32(next))
	return next
}

// setSpeed sets the speed by name; returns false for an unknown name.
func setSpeed(name string) bool {
	for s := speedLevel(0); s < numSpeedLevels; s++ {
		if s.String() == name {
			autoPilotSpeed.Store(int32(s))
			return true
		}
	}
	return false
}

func (s speedLevel) String() string {
	switch s {
	case speedInstant:
		return "instant"
	case speedFast:
		return "fast"
	case speedSlow:
		return "slow"
	default:
		return "normal"
	}
}

// delay returns a randomized thinking delay for the current speed.
func (s speedLevel) delay() time.Duration {
	switch s {
	case speedInstant:
		return 0
	case speedFast:
		return time.Duration(50+cryptoIntn(200)) * time.Millisecond
	case speedSlow:
		return time.Duration(1500+cryptoIntn(1500)) * time.Millisecond
	default:
		return time.Duration(500+cryptoIntn(1500)) * time.Millisecond
	}
}
