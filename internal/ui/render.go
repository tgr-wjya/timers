package ui

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"timers/internal/ipc"
	"timers/internal/timer"
)

var pieGlyphs = []string{"○", "◔", "◑", "◕", "●"}

func GlyphForProgress(p float64) string {
	if p <= 0 {
		return pieGlyphs[0]
	}
	if p >= 1.0 {
		return pieGlyphs[len(pieGlyphs)-1]
	}
	idx := int(p * float64(len(pieGlyphs)))
	if idx >= len(pieGlyphs) {
		idx = len(pieGlyphs) - 1
	}
	return pieGlyphs[idx]
}

func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Round(time.Second).Seconds())
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func FormatFilamentBar(p float64, barWidth int) string {
	if barWidth <= 0 {
		barWidth = 20
	}
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}

	filledLen := int(p * float64(barWidth))
	if filledLen > barWidth {
		filledLen = barWidth
	}

	var sb strings.Builder
	for i := 0; i < filledLen; i++ {
		sb.WriteString("━")
	}

	if filledLen < barWidth {
		sb.WriteString("╸")
		remaining := barWidth - filledLen - 1
		for i := 0; i < remaining; i++ {
			sb.WriteString("┈")
		}
	}

	return sb.String()
}

func FormatProgressLine(t *timer.Timer, now time.Time, barWidth int) string {
	p := t.Progress(now)
	glyph := GlyphForProgress(p)
	remStr := FormatDuration(t.Remaining(now))
	pct := int(p * 100)
	bar := FormatFilamentBar(p, barWidth)

	return fmt.Sprintf("  %s %s  %s  %s (%d%%)", glyph, remStr, bar, t.Title, pct)
}

func FormatWindowTitle(t *timer.Timer, now time.Time) string {
	p := t.Progress(now)
	glyph := GlyphForProgress(p)
	remStr := FormatDuration(t.Remaining(now))
	return fmt.Sprintf("\033]2;%s %s %s\007", glyph, remStr, t.Title)
}

func ResetWindowTitle() string {
	return "\033]2;\007"
}

func RenderWatch(client *ipc.Client, id int, out io.Writer) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	defer func() {
		fmt.Fprint(out, ResetWindowTitle())
	}()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Fprintf(out, "\nDetached from timer #%d. (Still running in background)\n", id)
			return nil

		case now := <-ticker.C:
			resp, err := client.Send(ipc.Request{Action: "list"})
			if err != nil {
				return fmt.Errorf("connection to daemon lost: %w", err)
			}

			var target *timer.Timer
			for _, tm := range resp.Timers {
				if tm.ID == id {
					target = tm
					break
				}
			}

			if target == nil {
				fmt.Fprintf(out, "\r\033[KTimer #%d not found or expired.\n", id)
				return nil
			}

			if target.Status == timer.StatusCompleted {
				fmt.Fprintf(out, "\r\033[K  ✔ 00:00  Timer #%d '%s' Completed!\n", target.ID, target.Title)
				return nil
			}

			if target.Status == timer.StatusCancelled {
				fmt.Fprintf(out, "\r\033[K  ✘ Timer #%d '%s' Cancelled.\n", target.ID, target.Title)
				return nil
			}

			// Render progress line and window title
			line := FormatProgressLine(target, now, 24)
			title := FormatWindowTitle(target, now)
			fmt.Fprintf(out, "\r\033[K%s%s", line, title)
		}
	}
}
