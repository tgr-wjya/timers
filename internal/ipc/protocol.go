package ipc

import (
	"timers/internal/timer"
)

type Request struct {
	Action      string   `json:"action"` // start, list, cancel, pause, resume, pomo_next, watch, ping, stop
	DurationSec int      `json:"duration_sec,omitempty"`
	Title       string   `json:"title,omitempty"`
	Category    string   `json:"category,omitempty"`
	ID          int      `json:"id,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	SinceSec    int      `json:"since_sec,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	Mute        bool     `json:"mute,omitempty"`
}

type Response struct {
	Status  string         `json:"status"` // "ok" or "error"
	Message string         `json:"message,omitempty"`
	Timers  []*timer.Timer `json:"timers,omitempty"`
	Timer   *timer.Timer   `json:"timer,omitempty"`
}
