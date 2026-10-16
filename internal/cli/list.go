package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"
	"timers/internal/ipc"
	"timers/internal/ui"
)

func RunList(args []string) error {
	client, err := getClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Send(ipc.Request{Action: "list"})
	if err != nil {
		return fmt.Errorf("failed to query timers: %w", err)
	}

	if len(resp.Timers) == 0 {
		fmt.Println("No active timers running.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tREMAINING\tSTATUS\tTARGET END")

	now := time.Now()
	for _, tm := range resp.Timers {
		remStr := ui.FormatDuration(tm.Remaining(now))
		endStr := tm.TargetEnd.Format("15:04:05")
		fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\n", tm.ID, tm.Title, remStr, string(tm.Status), endStr)
	}

	return w.Flush()
}
