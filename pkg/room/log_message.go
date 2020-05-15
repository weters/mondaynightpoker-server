package room

import (
	"mondaynightpoker-server/pkg/playable"
)

const logMessageLimit = 25

// addLogMessages adds a lot message
// Note: this must only be called from within the run loop
func (d *Dealer) addLogMessages(messages []*playable.LogMessage) {
	m := append(d.logMessages, messages...)
	count := len(m)
	if count > logMessageLimit {
		m = m[count-logMessageLimit:]
	}

	d.logMessages = m
}
