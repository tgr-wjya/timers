package pomodoro

import (
	"testing"
	"time"
)

func TestPomodoroCycle(t *testing.T) {
	cfg := Config{
		WorkDuration: 25 * time.Minute,
		ShortBreak:   5 * time.Minute,
		LongBreak:    15 * time.Minute,
		TotalRounds:  4,
	}

	c := NewCycle("Write Spec", cfg)

	if c.TaskName != "Write Spec" {
		t.Fatalf("expected task name 'Write Spec', got %q", c.TaskName)
	}
	if c.Round != 1 {
		t.Fatalf("expected round 1, got %d", c.Round)
	}
	if c.CurrentStage != StageWork {
		t.Fatalf("expected initial stage StageWork, got %v", c.CurrentStage)
	}
	if c.DurationForStage(c.CurrentStage) != 25*time.Minute {
		t.Fatalf("expected work duration 25m, got %v", c.DurationForStage(c.CurrentStage))
	}

	// Round 1: Work -> Short Break
	s := c.NextStage()
	if s != StageShortBreak || c.CurrentStage != StageShortBreak || c.Round != 1 {
		t.Fatalf("expected StageShortBreak round 1, got stage %v round %d", s, c.Round)
	}
	if c.DurationForStage(c.CurrentStage) != 5*time.Minute {
		t.Fatalf("expected short break duration 5m, got %v", c.DurationForStage(c.CurrentStage))
	}

	// Round 2: Short Break -> Work
	s = c.NextStage()
	if s != StageWork || c.CurrentStage != StageWork || c.Round != 2 {
		t.Fatalf("expected StageWork round 2, got stage %v round %d", s, c.Round)
	}

	// Round 2: Work -> Short Break
	s = c.NextStage()
	if s != StageShortBreak || c.Round != 2 {
		t.Fatalf("expected StageShortBreak round 2, got stage %v round %d", s, c.Round)
	}

	// Round 3: Short Break -> Work
	s = c.NextStage()
	if s != StageWork || c.Round != 3 {
		t.Fatalf("expected StageWork round 3, got stage %v round %d", s, c.Round)
	}

	// Round 3: Work -> Short Break
	s = c.NextStage()
	if s != StageShortBreak || c.Round != 3 {
		t.Fatalf("expected StageShortBreak round 3, got stage %v round %d", s, c.Round)
	}

	// Round 4: Short Break -> Work
	s = c.NextStage()
	if s != StageWork || c.Round != 4 {
		t.Fatalf("expected StageWork round 4, got stage %v round %d", s, c.Round)
	}

	// Round 4: Work -> Long Break
	s = c.NextStage()
	if s != StageLongBreak || c.CurrentStage != StageLongBreak || c.Round != 4 {
		t.Fatalf("expected StageLongBreak round 4, got stage %v round %d", s, c.Round)
	}
	if c.DurationForStage(c.CurrentStage) != 15*time.Minute {
		t.Fatalf("expected long break duration 15m, got %v", c.DurationForStage(c.CurrentStage))
	}

	// Long Break -> Completed
	s = c.NextStage()
	if s != StageCompleted || c.CurrentStage != StageCompleted {
		t.Fatalf("expected StageCompleted, got stage %v", s)
	}
}
