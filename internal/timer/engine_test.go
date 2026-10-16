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
