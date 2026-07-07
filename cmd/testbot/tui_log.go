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

// Window returns up to n entries in chronological order, ending offset
// entries before the newest. offset 0 is the live tail; offset is clamped
// so the window never scrolls past the oldest entry.
func (lb *LogBuffer) Window(n, offset int) []LogEntry {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if n <= 0 || n > lb.count {
		n = lb.count
	}
	maxOffset := lb.count - n
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}

	result := make([]LogEntry, n)
	startIdx := (lb.start + lb.count - n - offset) % lb.cap
	for i := range n {
		result[i] = lb.entries[(startIdx+i)%lb.cap]
	}
	return result
}

// RenderLog renders the last maxLines log entries for display.
func RenderLog(lb *LogBuffer, maxLines, width int) string {
	return RenderLogWindow(lb, maxLines, width, 0)
}

// RenderLogWindow renders maxLines log entries ending offset entries before
// the newest. When scrolled (offset > 0), the last line shows an indicator.
func RenderLogWindow(lb *LogBuffer, maxLines, width, offset int) string {
	entryLines := maxLines
	if offset > 0 && entryLines > 1 {
		entryLines-- // reserve a line for the scroll indicator
	}
	entries := lb.Window(entryLines, offset)
	if len(entries) == 0 {
		return styleLogEntry.Render("(no log entries)")
	}

	lines := make([]string, 0, len(entries)+1)
	for _, e := range entries {
		ts := e.Time.Format("15:04:05")
		msg := e.Message
		// Truncate to fit the available width (rune-safe)
		if maxMsg := width - len(ts) - 1; width > 0 {
			if r := []rune(msg); len(r) > maxMsg && maxMsg > 1 {
				msg = string(r[:maxMsg-1]) + "…"
			}
		}
		lines = append(lines, styleLogTime.Render(ts)+" "+styleLogEntry.Render(msg))
	}
	if offset > 0 {
		lines = append(lines, styleLogScroll.Render(fmt.Sprintf("── scrolled up %d · PgDn/End to follow ──", offset)))
	}
	return strings.Join(lines, "\n")
}
