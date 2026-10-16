package cli

import (
	"fmt"
	"strconv"
	"timers/internal/ipc"
)

func parseID(args []string, cmdName string) (int, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("timer ID required (e.g. 'timer %s 1')", cmdName)
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, fmt.Errorf("invalid timer ID %q: must be integer", args[0])
	}
	return id, nil
}

func RunPause(args []string) error {
	id, err := parseID(args, "pause")
	if err != nil {
		return err
	}

	client, err := getClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Send(ipc.Request{Action: "pause", ID: id})
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		return fmt.Errorf("error: %s", resp.Message)
	}

	fmt.Printf("Timer #%d '%s' paused.\n", id, resp.Timer.Title)
	return nil
}

func RunResume(args []string) error {
	id, err := parseID(args, "resume")
	if err != nil {
		return err
	}

	client, err := getClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Send(ipc.Request{Action: "resume", ID: id})
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		return fmt.Errorf("error: %s", resp.Message)
	}

	fmt.Printf("Timer #%d '%s' resumed.\n", id, resp.Timer.Title)
	return nil
}

func RunCancel(args []string) error {
	id, err := parseID(args, "cancel")
	if err != nil {
		return err
	}

	client, err := getClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Send(ipc.Request{Action: "cancel", ID: id})
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		return fmt.Errorf("error: %s", resp.Message)
	}

	fmt.Printf("Timer #%d cancelled.\n", id)
	return nil
}
