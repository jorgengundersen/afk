package harness

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestRunCmdExitCode0(t *testing.T) {
	cmd := exec.Command("true")
	exitCode, err := runCmd(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunCmdExitCode1(t *testing.T) {
	cmd := exec.Command("false")
	exitCode, err := runCmd(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunCmdContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")

	// Start a long-running process that writes its PID, then cancel context
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo $$ > %s; sleep 60", pidFile))
	done := make(chan struct{})
	var exitCode int
	var runErr error
	go func() {
		exitCode, runErr = runCmd(ctx, cmd)
		close(done)
	}()

	// Wait for process to start via PID file (no shared cmd.Process access)
	readPID(t, pidFile, 5*time.Second)

	cancel()
	<-done

	if runErr == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", runErr)
	}
	_ = exitCode
}

func TestRunCmdKillsGrandchildren(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "grandchild.pid")

	// The parent shell spawns a background grandchild (sleep), writes its PID,
	// then waits. When the group is killed, the grandchild should die too.
	script := fmt.Sprintf(`sh -c 'echo $$ > %s; sleep 60' &
wait`, pidFile)
	cmd := exec.Command("sh", "-c", script)

	done := make(chan struct{})
	go func() {
		runCmd(ctx, cmd)
		close(done)
	}()

	// Wait for grandchild PID file to appear
	grandchildPID := readPID(t, pidFile, 5*time.Second)

	cancel()
	<-done

	// Give the OS a moment to clean up
	time.Sleep(100 * time.Millisecond)

	// Verify the grandchild is dead: signal 0 should fail
	if processAlive(grandchildPID) {
		t.Fatalf("grandchild process %d is still alive after cancellation", grandchildPID)
	}
}

func TestRunCmdBinaryNotFound(t *testing.T) {
	cmd := exec.Command("nonexistent-binary-xyz-12345")
	_, err := runCmd(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestRunCmdSetsProcessGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")

	// Run a process that writes its PID, then verify it gets its own process group
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo $$ > %s; sleep 60", pidFile))
	done := make(chan struct{})
	go func() {
		runCmd(ctx, cmd)
		close(done)
	}()

	// Wait for process to start via PID file (no shared cmd.Process access)
	pid := readPID(t, pidFile, 5*time.Second)

	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		t.Fatalf("failed to get pgid: %v", err)
	}
	if pgid != pid {
		t.Fatalf("expected pgid %d == pid %d (own process group)", pgid, pid)
	}

	cancel()
	<-done
}
