package cli

import (
	"fmt"
	"strings"
	"timers/internal/ipc"
	"timers/internal/timer"
)

func RunPomodoro(args []string) error {
	client, err := getClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	if len(args) > 0 && args[0] == "next" {
		resp, err := client.Send(ipc.Request{Action: "pomo_next"})
		if err != nil {
			return err
		}
		if resp.Status != "ok" {
			return fmt.Errorf("error: %s", resp.Message)
		}
		if resp.Timer != nil {
			fmt.Printf("Pomodoro advanced: '%s' [%s] — alerts at %s\n",
				resp.Timer.Title, resp.Timer.Duration.String(), resp.Timer.TargetEnd.Format("15:04:05"))
		} else {
			fmt.Println(resp.Message)
		}
		return nil
	}

	taskName := "Pomodoro"
	var durationSec int

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--work" && i+1 < len(args) {
			dur, err := timer.ParseDuration(args[i+1])
			if err == nil {
				durationSec = int(dur.Seconds())
			}
			i++
		} else if !strings.HasPrefix(arg, "-") {
			taskName = arg
		}
	}

	resp, err := client.Send(ipc.Request{
		Action:      "pomo_start",
		Title:       taskName,
		DurationSec: durationSec,
	})
	if err != nil {
		return err
	}
	if resp.Status != "ok" || resp.Timer == nil {
		return fmt.Errorf("error: %s", resp.Message)
	}

	tm := resp.Timer
	fmt.Printf("Pomodoro started: '%s' [%s] — alerts at %s\n",
		tm.Title, tm.Duration.String(), tm.TargetEnd.Format("15:04:05"))
	fmt.Println("Run 'timer pomodoro next' when finished to advance to the next break/work interval.")

	return nil
}
