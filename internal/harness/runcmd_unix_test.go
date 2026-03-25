//go:build unix

package harness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

	// Start a long-running process, then cancel context
	cmd := exec.Command("sleep", "60")
	done := make(chan struct{})
	var exitCode int
	var runErr error
	go func() {
		exitCode, runErr = runCmd(ctx, cmd)
		close(done)
	}()

	// Wait for process to start
	for cmd.Process == nil {
		time.Sleep(10 * time.Millisecond)
	}

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

	// Parent shell spawns a grandchild that writes its PID to a temp file
	pidFile := fmt.Sprintf("/tmp/afk-test-grandchild-%d.pid", os.Getpid())
	defer os.Remove(pidFile)

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
	var grandchildPID int
	for i := 0; i < 100; i++ {
		data, err := os.ReadFile(pidFile)
		if err == nil && len(strings.TrimSpace(string(data))) > 0 {
			fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &grandchildPID)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if grandchildPID == 0 {
		t.Fatal("grandchild PID file never appeared")
	}

	cancel()
	<-done

	// Give the OS a moment to clean up
	time.Sleep(100 * time.Millisecond)

	// Verify the grandchild is dead: signal 0 should fail
	err := syscall.Kill(grandchildPID, 0)
	if err == nil {
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

	// Run a process and verify it gets its own process group (PGID = PID)
	cmd := exec.Command("sleep", "60")
	done := make(chan struct{})
	go func() {
		runCmd(ctx, cmd)
		close(done)
	}()

	// Wait for process to start
	for cmd.Process == nil {
		time.Sleep(10 * time.Millisecond)
	}

	pid := cmd.Process.Pid
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
