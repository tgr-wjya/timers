package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLICommands(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "timer_bin")
	runtimeDir := filepath.Join(tmpDir, "runtime")
	dataDir := filepath.Join(tmpDir, "data")

	// 1. Build binary
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v, output: %s", err, string(out))
	}

	runCLI := func(args ...string) (string, error) {
		cmd := exec.Command(binPath, args...)
		cmd.Env = append(os.Environ(),
			"XDG_RUNTIME_DIR="+runtimeDir,
			"XDG_DATA_HOME="+dataDir,
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// 2. Test --help
	out, err := runCLI("--help")
	if err != nil {
		t.Fatalf("failed to run --help: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "timer") {
		t.Fatalf("expected usage info in help, got: %s", out)
	}

	// 3. Test start timer
	out, err = runCLI("1s", "CLI Test")
	if err != nil {
		t.Fatalf("failed to start timer: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "CLI Test") || !strings.Contains(out, "started") {
		t.Fatalf("expected started confirmation, got: %s", out)
	}

	// 4. Test list
	out, err = runCLI("list")
	if err != nil {
		t.Fatalf("failed to list timers: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "CLI Test") {
		t.Fatalf("expected 'CLI Test' in list, got: %s", out)
	}

	// 5. Wait for timer to finish (1.5s)
	time.Sleep(1500 * time.Millisecond)

	// 6. Test log
	out, err = runCLI("log", "--today")
	if err != nil {
		t.Fatalf("failed to query log: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "CLI Test") {
		t.Fatalf("expected 'CLI Test' in log, got: %s", out)
	}

	// 7. Test stats
	out, err = runCLI("stats")
	if err != nil {
		t.Fatalf("failed to query stats: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "Focus Summary") {
		t.Fatalf("expected focus summary in stats, got: %s", out)
	}

	// 8. Test stop daemon
	out, err = runCLI("daemon", "stop")
	if err != nil {
		t.Fatalf("failed to stop daemon: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "shutting down") && !strings.Contains(out, "stopped") {
		t.Fatalf("expected daemon stop confirmation, got: %s", out)
	}
}
