package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// LogBuffer is a fixed-size ring buffer of log entries for the TUI.
type LogBuffer struct {
	mu      sync.Mutex
	entries []LogEntry
	cap     int
	start   int // index of oldest entry
	count   int // number of entries stored
}

// LogEntry is a single timestamped log line.
type LogEntry struct {
	Time    time.Time
	Message string
}

// NewLogBuffer creates a ring buffer that holds up to capacity entries.
func NewLogBuffer(capacity int) *LogBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &LogBuffer{
		entries: make([]LogEntry, capacity),
		cap:     capacity,
	}
}

// Add appends a new log entry, evicting the oldest if full.
func (lb *LogBuffer) Add(msg string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	idx := (lb.start + lb.count) % lb.cap
	lb.entries[idx] = LogEntry{
		Time:    time.Now(),
		Message: msg,
	}

	if lb.count < lb.cap {
		lb.count++
	} else {
		lb.start = (lb.start + 1) % lb.cap
	}
}

// Recent returns the last n entries in chronological order.
// If n <= 0 or n > count, returns all entries.
func (lb *LogBuffer) Recent(n int) []LogEntry {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if n <= 0 || n > lb.count {
		n = lb.count
	}

	result := make([]LogEntry, n)
	startIdx := (lb.start + lb.count - n) % lb.cap
	for i := range n {
		result[i] = lb.entries[(startIdx+i)%lb.cap]
	}
	return result
}

// Len returns the current number of entries.
func (lb *LogBuffer) Len() int {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.count
}

// RenderLog renders the last maxLines log entries for display.
func RenderLog(lb *LogBuffer, maxLines, width int) string {
	entries := lb.Recent(maxLines)
	if len(entries) == 0 {
		return styleLogEntry.Render("  (no log entries)")
	}

	lines := make([]string, len(entries))
	for i, e := range entries {
		ts := e.Time.Format("15:04:05")
		line := fmt.Sprintf("  %s  %s", ts, e.Message)
		// Truncate if wider than available width
		if width > 0 && len(line) > width-2 {
			line = line[:width-5] + "..."
		}
		lines[i] = styleLogEntry.Render(line)
	}
	return strings.Join(lines, "\n")
}
