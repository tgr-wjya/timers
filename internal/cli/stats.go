package cli

import (
	"fmt"
	"strings"
	"time"
	"timers/internal/daemon"
	"timers/internal/ipc"
	"timers/internal/ui"
)

func RunStats(args []string) error {
	store, err := getStore()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	since, _ := parseTimeFilter(args)
	stats, err := store.GetStats(since)
	if err != nil {
		return fmt.Errorf("failed to query stats: %w", err)
	}

	totalDur := time.Duration(stats.TotalFocusSec) * time.Second
	totalStr := ui.FormatDuration(totalDur)

	rate := 0
	if stats.TotalSessions > 0 {
		rate = int((float64(stats.CompletedSessions) / float64(stats.TotalSessions)) * 100)
	}

	fmt.Println("Focus Summary")
	fmt.Println(strings.Repeat("─", 45))
	fmt.Printf("Total Focus Time   : %s (%d sessions)\n", totalStr, stats.TotalSessions)
	fmt.Printf("Completed Sessions : %d (%d%% completion rate)\n", stats.CompletedSessions, rate)

	if len(stats.TopTasks) > 0 {
		fmt.Println("\nTop Tasks:")
		for i, task := range stats.TopTasks {
			taskDurStr := ui.FormatDuration(task.Duration)
			fmt.Printf("  %d. %s (%s, %d sessions)\n", i+1, task.Title, taskDurStr, task.Count)
		}
	}

	return nil
}

func RunDaemon(args []string) error {
	if len(args) > 0 && args[0] == "stop" {
		client, err := getClient()
		if err != nil {
			fmt.Println("Daemon is not running.")
			return nil
		}
		defer client.Close()

		resp, err := client.Send(ipc.Request{Action: "stop"})
		if err != nil {
			return err
		}
		fmt.Printf("Daemon: %s\n", resp.Message)
		return nil
	}

	if len(args) > 0 && args[0] == "status" {
		client, err := getClient()
		if err != nil || !client.Ping() {
			fmt.Println("Daemon status: inactive (will auto-spawn on next timer command).")
			return nil
		}
		defer client.Close()
		fmt.Println("Daemon status: active (running in background).")
		return nil
	}

	// Foreground daemon runner (invoked internally or manually: timer daemon [--socket s] [--db d])
	var socketPath, dbPath string
	for i := 0; i < len(args); i++ {
		if args[i] == "--socket" && i+1 < len(args) {
			socketPath = args[i+1]
			i++
		} else if args[i] == "--db" && i+1 < len(args) {
			dbPath = args[i+1]
			i++
		}
	}

	if socketPath == "" {
		socketPath = daemon.DefaultSocketPath()
	}
	if dbPath == "" {
		dbPath = daemon.DefaultDBPath()
	}

	d, err := daemon.NewDaemon(socketPath, dbPath)
	if err != nil {
		return err
	}
	return d.Start()
}
