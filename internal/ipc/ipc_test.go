package ipc

import (
	"path/filepath"
	"testing"
	"time"
	"timers/internal/timer"
)

func TestIPCRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	handler := func(req Request) Response {
		if req.Action == "ping" {
			return Response{Status: "ok", Message: "pong"}
		}
		if req.Action == "start" {
			return Response{
				Status: "ok",
				Timer: &timer.Timer{
					ID:       1,
					Title:    req.Title,
					Duration: time.Duration(req.DurationSec) * time.Second,
					Status:   timer.StatusRunning,
				},
			}
		}
		return Response{Status: "error", Message: "unknown action"}
	}

	server, err := NewServer(sockPath, handler)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer server.Close()

	client, err := Dial(sockPath)
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}
	defer client.Close()

	// 1. Verify ping
	if !client.Ping() {
		t.Fatal("expected ping to succeed")
	}

	// 2. Verify send request
	resp, err := client.Send(Request{
		Action:      "start",
		Title:       "Focus",
		DurationSec: 1500,
	})
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status ok, got %q (message: %s)", resp.Status, resp.Message)
	}
	if resp.Timer == nil || resp.Timer.Title != "Focus" {
		t.Fatalf("expected timer with title 'Focus', got %+v", resp.Timer)
	}

	// 3. Close server and verify Ping returns false
	_ = server.Close()
	if client.Ping() {
		t.Fatal("expected ping to fail after server closed")
	}
}
