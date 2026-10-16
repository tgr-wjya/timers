package ui

import (
	"strings"
	"testing"
	"time"
	"timers/internal/timer"
)

func TestFormatProgressLine(t *testing.T) {
	started := time.Now()
	dur := 10 * time.Minute

	// 0% elapsed
	tm0 := &timer.Timer{
		ID:        1,
		Title:     "Focus",
		Duration:  dur,
		StartedAt: started,
		TargetEnd: started.Add(dur),
		Status:    timer.StatusRunning,
	}
	line0 := FormatProgressLine(tm0, started, 20)
	if !strings.Contains(line0, "○") {
		t.Fatalf("expected glyph '○' for 0%% elapsed, got: %s", line0)
	}
	if !strings.Contains(line0, "0%") {
		t.Fatalf("expected '0%%' in line, got: %s", line0)
	}

	// 50% elapsed
	halfNow := started.Add(5 * time.Minute)
	lineHalf := FormatProgressLine(tm0, halfNow, 20)
	if !strings.Contains(lineHalf, "◑") {
		t.Fatalf("expected glyph '◑' for 50%% elapsed, got: %s", lineHalf)
	}
	if !strings.Contains(lineHalf, "50%") {
		t.Fatalf("expected '50%%' in line, got: %s", lineHalf)
	}

	// 100% elapsed
	fullNow := started.Add(10 * time.Minute)
	lineFull := FormatProgressLine(tm0, fullNow, 20)
	if !strings.Contains(lineFull, "●") {
		t.Fatalf("expected glyph '●' for 100%% elapsed, got: %s", lineFull)
	}
	if !strings.Contains(lineFull, "100%") {
		t.Fatalf("expected '100%%' in line, got: %s", lineFull)
	}
}

func TestFormatWindowTitle(t *testing.T) {
	started := time.Now()
	dur := 25 * time.Minute
	tm := &timer.Timer{
		ID:        1,
		Title:     "Deep Work",
		Duration:  dur,
		StartedAt: started,
		TargetEnd: started.Add(dur),
		Status:    timer.StatusRunning,
	}

	title := FormatWindowTitle(tm, started.Add(5*time.Minute))
	if !strings.HasPrefix(title, "\033]2;") || !strings.HasSuffix(title, "\007") {
		t.Fatalf("expected OSC 2 sequence, got %q", title)
	}
	if !strings.Contains(title, "Deep Work") {
		t.Fatalf("expected 'Deep Work' in window title, got %q", title)
	}
	if !strings.Contains(title, "20:00") {
		t.Fatalf("expected remaining time 20:00, got %q", title)
	}

	reset := ResetWindowTitle()
	if reset != "\033]2;\007" {
		t.Fatalf("expected reset sequence, got %q", reset)
	}
}
