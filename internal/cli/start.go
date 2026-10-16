package cli

import (
	"fmt"
	"os"
	"strings"
	"timers/internal/ipc"
	"timers/internal/timer"
	"timers/internal/ui"
)

func RunStart(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("duration required (e.g. 'timer 25m')")
	}

	durStr := args[0]
	duration, err := timer.ParseDuration(durStr)
	if err != nil {
		return err
	}

	title := ""
	watch := false
	mute := false
	var tags []string

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-f" || arg == "--watch":
			watch = true
		case arg == "-m" || arg == "--mute":
			mute = true
		case arg == "-t" || arg == "--tag":
			if i+1 < len(args) {
				tags = strings.Split(args[i+1], ",")
				i++
			}
		case strings.HasPrefix(arg, "-"):
			// Ignore other flags
		default:
			if title == "" {
				title = arg
			} else {
				title += " " + arg
			}
		}
	}

	client, err := getClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Send(ipc.Request{
		Action:      "start",
		Title:       title,
		DurationSec: int(duration.Seconds()),
		Tags:        tags,
		Mute:        mute,
	})
	if err != nil {
		return fmt.Errorf("failed to start timer: %w", err)
	}
	if resp.Status != "ok" || resp.Timer == nil {
		return fmt.Errorf("daemon error: %s", resp.Message)
	}

	tm := resp.Timer
	fmt.Printf("Timer #%d '%s' started [%s] — alerts at %s\n",
		tm.ID, tm.Title, tm.Duration.String(), tm.TargetEnd.Format("15:04:05"))

	if watch {
		return ui.RenderWatch(client, tm.ID, os.Stdout)
	}

	return nil
}

func RunWatch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("timer ID required (e.g. 'timer watch 1')")
	}
	var id int
	if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
		return fmt.Errorf("invalid timer ID %q: must be integer", args[0])
	}

	client, err := getClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	return ui.RenderWatch(client, id, os.Stdout)
}
