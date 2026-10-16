package pomodoro

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Stage string

const (
	StageWork       Stage = "work"
	StageShortBreak Stage = "short_break"
	StageLongBreak  Stage = "long_break"
	StageCompleted  Stage = "completed"
)

type Config struct {
	WorkDuration time.Duration `json:"work_duration"`
	ShortBreak   time.Duration `json:"short_break"`
	LongBreak    time.Duration `json:"long_break"`
	TotalRounds  int           `json:"total_rounds"`
}

func DefaultConfig() Config {
	return Config{
		WorkDuration: 25 * time.Minute,
		ShortBreak:   5 * time.Minute,
		LongBreak:    15 * time.Minute,
		TotalRounds:  4,
	}
}

type Cycle struct {
	ID           string    `json:"id"`
	TaskName     string    `json:"task_name"`
	Round        int       `json:"round"`
	Config       Config    `json:"config"`
	CurrentStage Stage     `json:"current_stage"`
	StartedAt    time.Time `json:"started_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewCycle(taskName string, cfg Config) *Cycle {
	if cfg.WorkDuration <= 0 {
		cfg.WorkDuration = 25 * time.Minute
	}
	if cfg.ShortBreak <= 0 {
		cfg.ShortBreak = 5 * time.Minute
	}
	if cfg.LongBreak <= 0 {
		cfg.LongBreak = 15 * time.Minute
	}
	if cfg.TotalRounds <= 0 {
		cfg.TotalRounds = 4
	}

	b := make([]byte, 8)
	rand.Read(b)
	id := hex.EncodeToString(b)

	now := time.Now()
	return &Cycle{
		ID:           id,
		TaskName:     taskName,
		Round:        1,
		Config:       cfg,
		CurrentStage: StageWork,
		StartedAt:    now,
		UpdatedAt:    now,
	}
}

func (c *Cycle) NextStage() Stage {
	c.UpdatedAt = time.Now()

	switch c.CurrentStage {
	case StageWork:
		if c.Round >= c.Config.TotalRounds {
			c.CurrentStage = StageLongBreak
		} else {
			c.CurrentStage = StageShortBreak
		}
	case StageShortBreak:
		c.Round++
		c.CurrentStage = StageWork
	case StageLongBreak:
		c.CurrentStage = StageCompleted
	case StageCompleted:
		// Already completed
	}

	return c.CurrentStage
}

func (c *Cycle) DurationForStage(s Stage) time.Duration {
	switch s {
	case StageWork:
		return c.Config.WorkDuration
	case StageShortBreak:
		return c.Config.ShortBreak
	case StageLongBreak:
		return c.Config.LongBreak
	default:
		return 0
	}
}
