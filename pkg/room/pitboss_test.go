package room

import (
	"testing"
	"time"

	"mondaynightpoker-server/pkg/model"
)

// TestPitBoss_survivesUnknownDisconnect ensures a disconnect for a table without a
// dealer (which legitimately happens when the dealer was already retired) does not
// terminate the pit boss run loop.
func TestPitBoss_survivesUnknownDisconnect(t *testing.T) {
	pb := NewPitBoss(testStore, PitBossOptions{StartGameDelay: time.Second})
	pb.StartShift()

	orphan := NewClient(nil, &model.Player{ID: 1}, &model.Table{UUID: "unknown-table"})
	pb.ClientDisconnected(orphan)

	// give the run loop time to process the disconnect before the connect arrives
	time.Sleep(250 * time.Millisecond)

	client := NewClient(nil, &model.Player{ID: 2}, &model.Table{UUID: "test-table"})
	pb.ClientConnected(client)

	select {
	case <-client.SendChan():
		// the run loop survived and dispatched the client to a dealer
	case <-time.After(5 * time.Second):
		t.Fatal("client never received a message; the pit boss run loop is dead")
	}
}
