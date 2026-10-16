package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"timers/internal/ipc"
)

func DefaultSocketPath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = filepath.Join(os.TempDir(), fmt.Sprintf("timer-%d", os.Getuid()))
	}
	return filepath.Join(runtimeDir, "timer.sock")
}

func DefaultDBPath() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "timer", "timers.db")
}

func EnsureDaemonRunning(socketPath, dbPath string) (*ipc.Client, error) {
	if socketPath == "" {
		socketPath = DefaultSocketPath()
	}
	if dbPath == "" {
		dbPath = DefaultDBPath()
	}

	// 1. Try to connect directly
	client, err := ipc.Dial(socketPath)
	if err == nil && client.Ping() {
		return client, nil
	}
	if client != nil {
		_ = client.Close()
	}

	// 2. Dead/stale socket cleanup
	_ = os.Remove(socketPath)

	// Ensure runtime dir exists
	_ = os.MkdirAll(filepath.Dir(socketPath), 0755)
	_ = os.MkdirAll(filepath.Dir(dbPath), 0755)

	// 3. Spawn detached daemon process
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("unable to determine binary path: %w", err)
	}

	cmd := exec.Command(exe, "daemon", "--socket", socketPath, "--db", dbPath)
	logPath := filepath.Join(filepath.Dir(socketPath), "daemon.log")
	logFile, logErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logErr == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		defer logFile.Close()
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Detach process group from terminal
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to auto-spawn daemon: %w", err)
	}

	// 4. Poll socket until ready
	var readyClient *ipc.Client
	for i := 0; i < 35; i++ {
		time.Sleep(30 * time.Millisecond)
		c, err := ipc.Dial(socketPath)
		if err == nil && c.Ping() {
			readyClient = c
			break
		}
	}

	if readyClient == nil {
		daemonLog, _ := os.ReadFile(logPath)
		return nil, fmt.Errorf("daemon started but socket %s was not ready in time (log: %s)", socketPath, string(daemonLog))
	}

	return readyClient, nil
}
