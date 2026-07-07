package playable

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCore_LogChan(t *testing.T) {
	c := NewCore()

	msgs := SimpleLogMessageSlice(1, "hello")
	c.SendLogMessages(msgs)

	select {
	case got := <-c.LogChan():
		assert.Equal(t, msgs, got)
	default:
		t.Fatal("expected a message on the log channel")
	}
}

func TestCore_SendLogMessage(t *testing.T) {
	c := NewCore()
	c.SendLogMessage(SimpleLogMessage(2, "single"))

	got := <-c.LogChan()
	assert.Len(t, got, 1)
	assert.Equal(t, "single", got[0].Message)
}

func TestCore_emptyAndNilSafety(t *testing.T) {
	c := NewCore()
	c.SendLogMessages(nil)
	c.SendLogMessages([]*LogMessage{})

	select {
	case <-c.LogChan():
		t.Fatal("empty sends must be dropped")
	default:
	}

	var nilCore *Core
	assert.Nil(t, nilCore.LogChan())
	nilCore.SendLogMessages(SimpleLogMessageSlice(1, "dropped")) // must not panic
	nilCore.SendLogMessage(SimpleLogMessage(1, "dropped"))       // must not panic
}

func TestCore_Interval(t *testing.T) {
	assert.Equal(t, time.Second, NewCore().Interval())
}
