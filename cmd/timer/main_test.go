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

	// 3. Test daemon status BEFORE starting: must be inactive and NOT auto-spawn!
	out, err = runCLI("daemon", "status")
	if err != nil {
		t.Fatalf("failed to run daemon status: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "inactive") {
		t.Fatalf("expected daemon status to be inactive, got: %s", out)
	}

	// 4. Test start timer with -m (mute)
	out, err = runCLI("1s", "CLI Test", "-m")
	if err != nil {
		t.Fatalf("failed to start timer: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "CLI Test") || !strings.Contains(out, "started") {
		t.Fatalf("expected started confirmation, got: %s", out)
	}

	// 5. Test daemon status AFTER starting: must now be active!
	out, err = runCLI("daemon", "status")
	if err != nil {
		t.Fatalf("failed to run daemon status: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "active") {
		t.Fatalf("expected daemon status to be active, got: %s", out)
	}

	// 6. Test start another and cancel all
	out, err = runCLI("10m", "Cancel Me")
	if err != nil {
		t.Fatalf("failed to start second timer: %v, output: %s", err, out)
	}
	out, err = runCLI("cancel", "all")
	if err != nil {
		t.Fatalf("failed to cancel all: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "All active timers cancelled") {
		t.Fatalf("expected cancel all confirmation, got: %s", out)
	}

	// 7. Test log
	out, err = runCLI("log", "--today")
	if err != nil {
		t.Fatalf("failed to query log: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "CLI Test") {
		t.Fatalf("expected 'CLI Test' in log, got: %s", out)
	}

	// 8. Test stats
	out, err = runCLI("stats")
	if err != nil {
		t.Fatalf("failed to query stats: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "Focus Summary") {
		t.Fatalf("expected focus summary in stats, got: %s", out)
	}

	// 9. Test stop daemon
	out, err = runCLI("daemon", "stop")
	if err != nil {
		t.Fatalf("failed to stop daemon: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "shutting down") && !strings.Contains(out, "stopped") {
		t.Fatalf("expected daemon stop confirmation, got: %s", out)
	}

	// 10. Verify status is inactive after stop
	time.Sleep(150 * time.Millisecond)
	out, _ = runCLI("daemon", "status")
	if !strings.Contains(out, "inactive") {
		t.Fatalf("expected status inactive after stop, got: %s", out)
	}
}
