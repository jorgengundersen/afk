//go:build unix

package harness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestRunAgentUsesRunCmd verifies that runAgent delegates to runCmd by
// checking that context cancellation returns context.Canceled — the
// signature behaviour of runCmd's process group cleanup path.
// Without runCmd, exec.CommandContext returns a generic exec error.
func TestRunAgentUsesRunCmd(t *testing.T) {
	// Create a helper script that ignores arguments and sleeps.
	dir := t.TempDir()
	helper := filepath.Join(dir, "sleeper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nsleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	var runErr error
	go func() {
		_, runErr = runAgent(ctx, helper, "ignored", "", "")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if runErr == nil {
		t.Fatal("expected error for cancelled context")
	}
	if runErr != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", runErr)
	}
}

// TestRawRunKillsGrandchildren verifies that Raw.Run delegates to runCmd
// with process group management, ensuring grandchild cleanup on cancellation.
func TestRawRunKillsGrandchildren(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pidFile := t.TempDir() + "/grandchild.pid"
	script := fmt.Sprintf("sh -c 'echo $$ > %s; sleep 60' & wait", pidFile)

	r := &Raw{template: script}
	done := make(chan struct{})
	go func() {
		r.Run(ctx, "ignored")
		close(done)
	}()

	var grandchildPID int
	for i := 0; i < 100; i++ {
		data, err := os.ReadFile(pidFile)
		if err == nil && len(strings.TrimSpace(string(data))) > 0 {
			fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &grandchildPID)
			if grandchildPID > 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if grandchildPID == 0 {
		t.Fatal("grandchild PID file never appeared")
	}

	cancel()
	<-done

	time.Sleep(100 * time.Millisecond)

	err := syscall.Kill(grandchildPID, 0)
	if err == nil {
		t.Fatalf("grandchild process %d still alive after cancellation", grandchildPID)
	}
}
