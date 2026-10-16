package timer

import (
	"time"
)

type Status string

const (
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

type Category string

const (
	CategoryGeneral       Category = "general"
	CategoryPomodoroWork  Category = "pomodoro_work"
	CategoryPomodoroBreak Category = "pomodoro_break"
)

type Timer struct {
	ID                  int           `json:"id"`
	Title               string        `json:"title"`
	Category            Category      `json:"category"`
	Duration            time.Duration `json:"duration"`
	StartedAt           time.Time     `json:"started_at"`
	TargetEnd           time.Time     `json:"target_end"`
	EndedAt             *time.Time    `json:"ended_at,omitempty"`
	Status              Status        `json:"status"`
	RemainingWhenPaused time.Duration `json:"remaining_when_paused,omitempty"`
	Tags                []string      `json:"tags,omitempty"`
}

func (t *Timer) Remaining(now time.Time) time.Duration {
	if t.Status == StatusPaused {
		return t.RemainingWhenPaused
	}
	if t.Status == StatusCompleted || t.Status == StatusCancelled {
		return 0
	}
	rem := t.TargetEnd.Sub(now)
	if rem < 0 {
		return 0
	}
	return rem
}

func (t *Timer) Progress(now time.Time) float64 {
	if t.Duration <= 0 {
		return 1.0
	}
	rem := t.Remaining(now)
	elapsed := t.Duration - rem
	p := float64(elapsed) / float64(t.Duration)
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}
