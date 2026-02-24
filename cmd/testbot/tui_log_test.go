package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogBufferBasic(t *testing.T) {
	lb := NewLogBuffer(5)
	assert.Equal(t, 0, lb.Len())

	lb.Add("hello")
	assert.Equal(t, 1, lb.Len())

	entries := lb.Recent(10)
	assert.Len(t, entries, 1)
	assert.Equal(t, "hello", entries[0].Message)
}

func TestLogBufferWrap(t *testing.T) {
	lb := NewLogBuffer(3)

	lb.Add("a")
	lb.Add("b")
	lb.Add("c")
	assert.Equal(t, 3, lb.Len())

	// Adding a 4th should evict "a"
	lb.Add("d")
	assert.Equal(t, 3, lb.Len())

	entries := lb.Recent(10)
	assert.Len(t, entries, 3)
	assert.Equal(t, "b", entries[0].Message)
	assert.Equal(t, "c", entries[1].Message)
	assert.Equal(t, "d", entries[2].Message)
}

func TestLogBufferRecentN(t *testing.T) {
	lb := NewLogBuffer(10)
	for i := range 5 {
		lb.Add(fmt.Sprintf("msg%d", i))
	}

	entries := lb.Recent(2)
	assert.Len(t, entries, 2)
	assert.Equal(t, "msg3", entries[0].Message)
	assert.Equal(t, "msg4", entries[1].Message)
}

func TestLogBufferRecentZero(t *testing.T) {
	lb := NewLogBuffer(5)
	lb.Add("a")
	lb.Add("b")

	entries := lb.Recent(0)
	assert.Len(t, entries, 2)
}

func TestNewLogBufferMinCap(t *testing.T) {
	lb := NewLogBuffer(0)
	assert.Equal(t, 1, lb.cap)

	lb.Add("x")
	assert.Equal(t, 1, lb.Len())

	lb.Add("y")
	assert.Equal(t, 1, lb.Len())
	entries := lb.Recent(10)
	assert.Equal(t, "y", entries[0].Message)
}

func TestRenderLogEmpty(t *testing.T) {
	lb := NewLogBuffer(5)
	result := RenderLog(lb, 5, 80)
	assert.Contains(t, result, "no log entries")
}

func TestRenderLogEntries(t *testing.T) {
	lb := NewLogBuffer(10)
	lb.Add("test message")
	lb.Add("another message")

	result := RenderLog(lb, 5, 80)
	assert.Contains(t, result, "test message")
	assert.Contains(t, result, "another message")
}
