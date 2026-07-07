package playable

import "time"

// Core provides the plumbing every game needs: the buffered log channel the dealer
// consumes and the shared tick interval. Embed *Core in a game struct; the methods
// are safe on a nil receiver so struct-literal test fixtures keep working.
type Core struct {
	logChan chan []*LogMessage
}

// NewCore returns an initialized Core
func NewCore() *Core {
	return &Core{
		logChan: make(chan []*LogMessage, 256),
	}
}

// LogChan returns a read-only channel of log messages
func (c *Core) LogChan() <-chan []*LogMessage {
	if c == nil {
		return nil
	}

	return c.logChan
}

// SendLogMessages delivers log messages to the dealer
// No-op on a nil Core or an empty message list.
func (c *Core) SendLogMessages(msgs []*LogMessage) {
	if c == nil || len(msgs) == 0 {
		return
	}

	c.logChan <- msgs
}

// SendLogMessage delivers a single log message to the dealer
func (c *Core) SendLogMessage(msg *LogMessage) {
	c.SendLogMessages([]*LogMessage{msg})
}

// Interval is how often the dealer should call Tick() on the game
func (c *Core) Interval() time.Duration {
	return time.Second
}
