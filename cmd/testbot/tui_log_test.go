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

func TestLogBufferWindow(t *testing.T) {
	lb := NewLogBuffer(10)
	for i := range 8 {
		lb.Add(fmt.Sprintf("msg%d", i))
	}

	t.Run("offset zero is the live tail", func(t *testing.T) {
		entries := lb.Window(3, 0)
		assert.Len(t, entries, 3)
		assert.Equal(t, "msg5", entries[0].Message)
		assert.Equal(t, "msg7", entries[2].Message)
	})

	t.Run("offset scrolls back", func(t *testing.T) {
		entries := lb.Window(3, 2)
		assert.Len(t, entries, 3)
		assert.Equal(t, "msg3", entries[0].Message)
		assert.Equal(t, "msg5", entries[2].Message)
	})

	t.Run("offset clamps at oldest entry", func(t *testing.T) {
		entries := lb.Window(3, 100)
		assert.Len(t, entries, 3)
		assert.Equal(t, "msg0", entries[0].Message)
		assert.Equal(t, "msg2", entries[2].Message)
	})

	t.Run("negative offset behaves like zero", func(t *testing.T) {
		entries := lb.Window(3, -5)
		assert.Equal(t, "msg7", entries[2].Message)
	})

	t.Run("window after wrap-around", func(t *testing.T) {
		small := NewLogBuffer(3)
		for i := range 5 {
			small.Add(fmt.Sprintf("m%d", i))
		}
		entries := small.Window(2, 1)
		assert.Len(t, entries, 2)
		assert.Equal(t, "m2", entries[0].Message)
		assert.Equal(t, "m3", entries[1].Message)
	})
}

func TestRenderLogWindowScrollIndicator(t *testing.T) {
	lb := NewLogBuffer(10)
	for i := range 8 {
		lb.Add(fmt.Sprintf("msg%d", i))
	}

	result := RenderLogWindow(lb, 4, 80, 2)
	assert.Contains(t, result, "scrolled up 2")

	result = RenderLogWindow(lb, 4, 80, 0)
	assert.NotContains(t, result, "scrolled")
}
