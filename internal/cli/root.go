package cli

import (
	"fmt"
	"strings"
	"time"
	"timers/internal/daemon"
	"timers/internal/ipc"
	"timers/internal/storage"
	"timers/internal/timer"
)

const HelpText = `Usage:
  timer <duration> [title] [flags]       Start a new timer (e.g. timer 25m "Work", timer 1h30m)
  timer list                             List all running and paused timers (alias: ls)
  timer pause <id>                       Pause a running timer
  timer resume <id>                      Resume a paused timer
  timer cancel <id>                      Cancel an active timer
  timer watch <id>                       Watch live timer countdown in terminal
  timer pomodoro [task] [flags]          Start a Pomodoro cycle (alias: pomo)
  timer log [flags]                      View completed timer sessions (alias: history)
  timer stats                            View focus time summary and statistics
  timer daemon [start|stop|status]       Manage background timer daemon

Flags:
  -f, --watch     Attach and watch live countdown in foreground
  -m, --mute      Disable audio chime for this timer
  -t, --tag       Add comma-separated tags (e.g. -t coding,client)
  -h, --help      Show this help message
`

func Execute(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Print(HelpText)
		return nil
	}

	cmd := args[0]

	// Handle explicit subcommands
	switch cmd {
	case "list", "ls":
		return RunList(args[1:])
	case "pause":
		return RunPause(args[1:])
	case "resume":
		return RunResume(args[1:])
	case "cancel":
		return RunCancel(args[1:])
	case "watch":
		return RunWatch(args[1:])
	case "pomodoro", "pomo":
		return RunPomodoro(args[1:])
	case "log", "history":
		return RunLog(args[1:])
	case "stats":
		return RunStats(args[1:])
	case "daemon":
		return RunDaemon(args[1:])
	default:
		// Attempt duration shorthand: e.g. "timer 25m 'Focus'"
		_, err := timer.ParseDuration(cmd)
		if err == nil {
			return RunStart(args)
		}
		// If starts with digits or looks like a duration, show parsing error
		if strings.IndexAny(cmd, "0123456789") == 0 {
			return err
		}
		return fmt.Errorf("unknown command %q\nRun 'timer --help' for usage.", cmd)
	}
}

func getClient() (*ipc.Client, error) {
	return daemon.EnsureDaemonRunning("", "")
}

func getStore() (*storage.Store, error) {
	dbPath := daemon.DefaultDBPath()
	return storage.NewStore(dbPath)
}

func parseTimeFilter(args []string) (time.Time, int) {
	since := time.Now().Add(-24 * time.Hour) // default: today
	limit := 30

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--today":
			now := time.Now()
			since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case "--yesterday":
			now := time.Now().Add(-24 * time.Hour)
			since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case "--week":
			since = time.Now().Add(-7 * 24 * time.Hour)
		case "--since":
			if i+1 < len(args) {
				dur, err := timer.ParseDuration(args[i+1])
				if err == nil {
					since = time.Now().Add(-dur)
				}
				i++
			}
		}
	}

	return since, limit
}
