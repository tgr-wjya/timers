package timer

import (
	"sync"
	"testing"
	"time"
)

func TestEngineLifecycle(t *testing.T) {
	var completed []*Timer
	var mu sync.Mutex

	cb := func(tm *Timer, event string) {
		mu.Lock()
		defer mu.Unlock()
		if event == "completed" {
			completed = append(completed, tm)
		}
	}

	engine := NewEngine(cb)
	defer engine.Stop()

	// 1. Start a timer
	tm := engine.Start("Deep Work", 10*time.Second, CategoryGeneral, []string{"focus"})
	if tm.ID != 1 {
		t.Fatalf("expected ID 1, got %d", tm.ID)
	}
	if tm.Status != StatusRunning {
		t.Fatalf("expected StatusRunning, got %v", tm.Status)
	}
	expectedEnd := tm.StartedAt.Add(10 * time.Second)
	if !tm.TargetEnd.Equal(expectedEnd) {
		t.Fatalf("expected TargetEnd %v, got %v", expectedEnd, tm.TargetEnd)
	}

	// 2. Pause the timer
	time.Sleep(50 * time.Millisecond)
	paused, err := engine.Pause(tm.ID)
	if err != nil {
		t.Fatalf("unexpected error pausing: %v", err)
	}
	if paused.Status != StatusPaused {
		t.Fatalf("expected StatusPaused, got %v", paused.Status)
	}
	if paused.RemainingWhenPaused <= 0 || paused.RemainingWhenPaused > 10*time.Second {
		t.Fatalf("unexpected RemainingWhenPaused: %v", paused.RemainingWhenPaused)
	}

	// 3. Resume the timer
	time.Sleep(50 * time.Millisecond)
	resumed, err := engine.Resume(tm.ID)
	if err != nil {
		t.Fatalf("unexpected error resuming: %v", err)
	}
	if resumed.Status != StatusRunning {
		t.Fatalf("expected StatusRunning, got %v", resumed.Status)
	}

	// 4. Tick past target end
	future := resumed.TargetEnd.Add(1 * time.Second)
	expired := engine.Tick(future)
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired timer, got %d", len(expired))
	}
	if expired[0].Status != StatusCompleted {
		t.Fatalf("expected StatusCompleted, got %v", expired[0].Status)
	}

	mu.Lock()
	if len(completed) != 1 {
		t.Fatalf("expected 1 callback completed, got %d", len(completed))
	}
	mu.Unlock()

	// 5. Start another and cancel it
	tm2 := engine.Start("Tea", 5*time.Minute, CategoryGeneral, nil)
	cancelled, err := engine.Cancel(tm2.ID)
	if err != nil {
		t.Fatalf("unexpected error cancelling: %v", err)
	}
	if cancelled.Status != StatusCancelled {
		t.Fatalf("expected StatusCancelled, got %v", cancelled.Status)
	}
}

func TestEngineRestoreAndIDSequence(t *testing.T) {
	engine := NewEngine(nil)
	defer engine.Stop()

	// Simulate restoring timers #5 and #12 from SQLite
	existing := []*Timer{
		{ID: 5, Title: "Existing 5", Status: StatusRunning, TargetEnd: time.Now().Add(10 * time.Minute)},
		{ID: 12, Title: "Existing 12", Status: StatusPaused, RemainingWhenPaused: 5 * time.Minute},
	}

	engine.Restore(existing)

	// Verify both timers exist in engine
	t5, ok5 := engine.Get(5)
	if !ok5 || t5.Title != "Existing 5" {
		t.Fatalf("expected timer 5 restored, got %+v", t5)
	}
	t12, ok12 := engine.Get(12)
	if !ok12 || t12.Title != "Existing 12" {
		t.Fatalf("expected timer 12 restored, got %+v", t12)
	}

	// Starting a new timer must assign ID 13 (max + 1), avoiding collision with 1..12
	next := engine.Start("New Task", 10*time.Minute, CategoryGeneral, nil)
	if next.ID != 13 {
		t.Fatalf("expected next ID to be 13, got %d", next.ID)
	}
}

