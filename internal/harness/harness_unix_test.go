//go:build unix

package harness

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestClaudeRunMultipleIterations verifies that Run() can be called more than
// once on the same Claude instance. Before the fix, the second call wrote to a
// closed pipe and silently lost all subprocess output.
func TestClaudeRunMultipleIterations(t *testing.T) {
	dir := t.TempDir()
	helper := filepath.Join(dir, "claude")
	script := `#!/bin/sh
echo '{"type":"result","cost_usd":0,"duration_ms":0,"result":"ok","is_error":false}'
`
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	var buf bytes.Buffer
	r, err := New("claude", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	c := r.(*Claude)
	c.output = &buf

	// First run should succeed.
	code, err := c.Run(context.Background(), "test1")
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}
	if code != 0 {
		t.Fatalf("first run exit code: %d", code)
	}

	// Second run must also succeed — this is the bug.
	buf.Reset()
	code, err = c.Run(context.Background(), "test2")
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}
	if code != 0 {
		t.Fatalf("second run exit code: %d", code)
	}
}

// TestRunCmdContextCancelReturnsContextCanceled verifies that runCmd's
// process group cleanup path returns context.Canceled on cancellation.
func TestRunCmdContextCancelReturnsContextCanceled(t *testing.T) {
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
		cmd := exec.CommandContext(ctx, helper)
		_, runErr = runCmd(ctx, cmd)
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
