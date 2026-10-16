package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"timers/internal/ui"
)

func RunLog(args []string) error {
	store, err := getStore()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	since, limit := parseTimeFilter(args)
	history, err := store.GetHistory(since, limit)
	if err != nil {
		return fmt.Errorf("failed to fetch history: %w", err)
	}

	if len(history) == 0 {
		fmt.Println("No past timer sessions found for this timeframe.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tDATE\tTIME\tDURATION\tTITLE\tSTATUS")

	for _, tm := range history {
		dateStr := tm.StartedAt.Format("2006-01-02")
		timeStr := tm.StartedAt.Format("15:04")
		durStr := ui.FormatDuration(tm.Duration)
		fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\t%s\n",
			tm.ID, dateStr, timeStr, durStr, tm.Title, string(tm.Status))
	}

	return w.Flush()
}
