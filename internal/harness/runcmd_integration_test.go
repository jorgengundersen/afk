//go:build unix

package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// helperScript creates an executable shell script in dir and returns its path.
func helperScript(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// readPID polls for a PID file to appear and returns the PID.
func readPID(t *testing.T, path string, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			s := strings.TrimSpace(string(data))
			if pid, err := strconv.Atoi(s); err == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("PID file %s never appeared within %v", path, timeout)
	return 0
}

// processAlive checks whether a process is still running.
func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// TestIntegrationGrandchildKilledOnCancel spawns a helper script that forks a
// grandchild. Both PIDs are written to temp files. After context cancellation,
// both parent and grandchild must be dead.
func TestIntegrationGrandchildKilledOnCancel(t *testing.T) {
	dir := t.TempDir()
	parentPIDFile := filepath.Join(dir, "parent.pid")
	grandchildPIDFile := filepath.Join(dir, "grandchild.pid")

	script := helperScript(t, dir, "spawn.sh", fmt.Sprintf(`#!/bin/sh
echo $$ > %s
sh -c 'echo $$ > %s; sleep 300' &
wait
`, parentPIDFile, grandchildPIDFile))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runCmd(ctx, exec.Command(script))
		close(done)
	}()

	parentPID := readPID(t, parentPIDFile, 5*time.Second)
	grandchildPID := readPID(t, grandchildPIDFile, 5*time.Second)

	cancel()
	<-done

	// Give the OS a moment to reap
	time.Sleep(100 * time.Millisecond)

	if processAlive(parentPID) {
		t.Errorf("parent process %d still alive after cancellation", parentPID)
	}
	if processAlive(grandchildPID) {
		t.Errorf("grandchild process %d still alive after cancellation", grandchildPID)
	}
}

// TestIntegrationSIGTERMBeforeSIGKILL verifies that runCmd sends SIGTERM first.
// The helper script traps SIGTERM, writes a marker file, then exits. If only
// SIGKILL were sent, the trap would never fire and the marker would be absent.
func TestIntegrationSIGTERMBeforeSIGKILL(t *testing.T) {
	dir := t.TempDir()
	markerFile := filepath.Join(dir, "got-sigterm")

	pidFile := filepath.Join(dir, "pid")

	script := helperScript(t, dir, "trap-term.sh", fmt.Sprintf(`#!/bin/sh
trap 'echo yes > %s; exit 0' TERM
echo $$ > %s
sleep 300
`, markerFile, pidFile))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runCmd(ctx, exec.Command(script))
		close(done)
	}()

	// Wait for the process to start and register its trap via PID file
	readPID(t, pidFile, 5*time.Second)
	time.Sleep(50 * time.Millisecond) // allow trap to register

	cancel()
	<-done

	// The trap handler should have written the marker
	data, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("SIGTERM marker file not created — process was likely killed with SIGKILL, not SIGTERM: %v", err)
	}
	if strings.TrimSpace(string(data)) != "yes" {
		t.Fatalf("unexpected marker content: %q", string(data))
	}
}

// TestIntegrationNormalExitNoSignals verifies that a process tree that exits
// on its own returns the correct exit code and no error, without any signal
// being sent. The helper script spawns a grandchild that also exits quickly.
// A SIGTERM trap writes a marker — its absence proves no signal was sent.
func TestIntegrationNormalExitNoSignals(t *testing.T) {
	dir := t.TempDir()
	markerFile := filepath.Join(dir, "got-signal")
	resultFile := filepath.Join(dir, "result")

	script := helperScript(t, dir, "clean-exit.sh", fmt.Sprintf(`#!/bin/sh
trap 'echo yes > %s' TERM INT
sh -c 'echo grandchild-done > %s' &
wait
exit 0
`, markerFile, resultFile))

	exitCode, err := runCmd(context.Background(), exec.Command(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	// Grandchild should have completed its work
	data, err := os.ReadFile(resultFile)
	if err != nil {
		t.Fatalf("grandchild result file not found: %v", err)
	}
	if strings.TrimSpace(string(data)) != "grandchild-done" {
		t.Fatalf("unexpected result: %q", string(data))
	}

	// No signal should have been sent
	if _, err := os.Stat(markerFile); err == nil {
		t.Fatal("signal marker exists — a signal was sent during normal exit")
	}
}
