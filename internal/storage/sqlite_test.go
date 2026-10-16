package storage

import (
	"testing"
	"time"
	"timers/internal/timer"
)

func TestStorageOperations(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to open memory sqlite: %v", err)
	}
	defer store.Close()

	now := time.Now()
	tm := &timer.Timer{
		ID:        1,
		Title:     "Deep Work",
		Category:  timer.CategoryGeneral,
		Duration:  25 * time.Minute,
		StartedAt: now,
		TargetEnd: now.Add(25 * time.Minute),
		Status:    timer.StatusRunning,
		Tags:      []string{"coding", "spec"},
	}

	// 1. Save active timer
	if err := store.SaveTimer(tm); err != nil {
		t.Fatalf("failed to save timer: %v", err)
	}

	// 2. Fetch active timers
	active, err := store.GetActiveTimers()
	if err != nil {
		t.Fatalf("failed to get active timers: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active timer, got %d", len(active))
	}
	if active[0].Title != "Deep Work" {
		t.Fatalf("expected title 'Deep Work', got %q", active[0].Title)
	}

	// 3. Update timer to completed
	ended := now.Add(25 * time.Minute)
	tm.Status = timer.StatusCompleted
	tm.EndedAt = &ended
	if err := store.UpdateTimer(tm); err != nil {
		t.Fatalf("failed to update timer: %v", err)
	}

	// Active should now be 0
	activeAfter, err := store.GetActiveTimers()
	if err != nil {
		t.Fatalf("failed to get active timers after update: %v", err)
	}
	if len(activeAfter) != 0 {
		t.Fatalf("expected 0 active timers, got %d", len(activeAfter))
	}

	// 4. Fetch history
	history, err := store.GetHistory(now.Add(-1*time.Hour), 10)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if history[0].Status != timer.StatusCompleted {
		t.Fatalf("expected status completed, got %v", history[0].Status)
	}

	// 5. Fetch stats
	stats, err := store.GetStats(now.Add(-1 * time.Hour))
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.TotalSessions != 1 {
		t.Fatalf("expected 1 total session, got %d", stats.TotalSessions)
	}
	if stats.CompletedSessions != 1 {
		t.Fatalf("expected 1 completed session, got %d", stats.CompletedSessions)
	}
	if stats.TotalFocusSec != 25*60 {
		t.Fatalf("expected 1500 focus seconds, got %d", stats.TotalFocusSec)
	}
}
