package daemon

import (
	"path/filepath"
	"testing"
	"time"
	"timers/internal/ipc"
	"timers/internal/storage"
)

func TestDaemonLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test_daemon.sock")
	dbPath := filepath.Join(tmpDir, "test_timers.db")

	d, err := NewDaemon(sockPath, dbPath)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- d.Start()
	}()

	// Wait for daemon to become ready
	var client *ipc.Client
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		c, err := ipc.Dial(sockPath)
		if err == nil && c.Ping() {
			client = c
			break
		}
	}
	if client == nil {
		t.Fatal("failed to connect to daemon within timeout")
	}
	defer client.Close()

	// 1. Start a fast timer (150ms)
	resp, err := client.Send(ipc.Request{
		Action:      "start",
		Title:       "Quick Test",
		DurationSec: 1, // 1s
	})
	if err != nil || resp.Status != "ok" || resp.Timer == nil {
		t.Fatalf("failed to start timer: err=%v, resp=%+v", err, resp)
	}
	timerID := resp.Timer.ID

	// 2. Query active timers
	listResp, err := client.Send(ipc.Request{Action: "list"})
	if err != nil || listResp.Status != "ok" {
		t.Fatalf("failed to list timers: %v", err)
	}
	if len(listResp.Timers) != 1 {
		t.Fatalf("expected 1 active timer, got %d", len(listResp.Timers))
	}

	// 3. Pause and Resume
	pauseResp, err := client.Send(ipc.Request{Action: "pause", ID: timerID})
	if err != nil || pauseResp.Status != "ok" {
		t.Fatalf("failed to pause timer: %v", err)
	}
	resumeResp, err := client.Send(ipc.Request{Action: "resume", ID: timerID})
	if err != nil || resumeResp.Status != "ok" {
		t.Fatalf("failed to resume timer: %v", err)
	}

	// 4. Stop the daemon cleanly
	stopResp, err := client.Send(ipc.Request{Action: "stop"})
	if err != nil || stopResp.Status != "ok" {
		t.Fatalf("failed to stop daemon: %v", err)
	}

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("daemon exited with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("daemon did not exit within timeout")
	}

	// 5. Verify records in database
	store, err := storage.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open database to verify: %v", err)
	}
	defer store.Close()

	history, err := store.GetHistory(time.Now().Add(-1*time.Hour), 10)
	if err != nil {
		t.Fatalf("failed to query history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 saved timer in history, got %d", len(history))
	}
}
