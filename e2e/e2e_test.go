//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// afkBin is the path to the compiled afk binary, set by TestMain.
var afkBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "afk-e2e-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	afkBin = filepath.Join(dir, "afk")
	build := exec.Command("go", "build", "-race", "-o", afkBin, "../cmd/afk")
	if out, err := build.CombinedOutput(); err != nil {
		panic("go build failed: " + string(out))
	}

	os.Exit(m.Run())
}

// fakeClaude creates a temp dir with a "claude" script that exits with the
// given code and optionally echoes its arguments. Returns the dir path.
func fakeClaude(t *testing.T, exitCode int) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	content := "#!/bin/sh\necho \"$@\"\nexit " + string(rune('0'+exitCode)) + "\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestHelp_PrintsUsage(t *testing.T) {
	cmd := exec.Command(afkBin, "-help")
	out, _ := cmd.CombinedOutput()

	output := string(out)
	if !strings.Contains(output, "Usage") && !strings.Contains(output, "-prompt") {
		t.Fatalf("expected usage output, got:\n%s", output)
	}

	if code := cmd.ProcessState.ExitCode(); code != 2 {
		t.Fatalf("expected exit code 2 for -help, got %d", code)
	}
}

func TestUnknownFlag_Exit2(t *testing.T) {
	cmd := exec.Command(afkBin, "--bogus-flag")
	out, _ := cmd.CombinedOutput()

	if code := cmd.ProcessState.ExitCode(); code != 2 {
		t.Fatalf("expected exit code 2, got %d\noutput: %s", code, out)
	}
}

func TestMaxIter1_FakeHarness_Exit0(t *testing.T) {
	dir := fakeClaude(t, 0)
	cmd := exec.Command(afkBin, "-n", "1", "-prompt", "hello")
	cmd.Env = append(os.Environ(), "PATH="+dir)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if code := cmd.ProcessState.ExitCode(); code != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", code, out)
	}
}

func TestLogFile_CreatedWithExpectedEvents(t *testing.T) {
	dir := fakeClaude(t, 0)
	logDir := t.TempDir()

	cmd := exec.Command(afkBin, "-n", "1", "-prompt", "hello", "-log", logDir)
	cmd.Env = append(os.Environ(), "PATH="+dir)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Verify a log file was created.
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("failed to read log dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected log file to be created, but log dir is empty")
	}

	logFile := filepath.Join(logDir, entries[0].Name())
	if !strings.HasPrefix(entries[0].Name(), "afk-") || !strings.HasSuffix(entries[0].Name(), ".log") {
		t.Fatalf("unexpected log file name: %s", entries[0].Name())
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	logContent := string(content)

	// Verify expected events are present.
	for _, event := range []string{"loop_start", "iteration_start", "iteration_end", "loop_end"} {
		if !strings.Contains(logContent, event) {
			t.Errorf("log missing expected event %q\nlog content:\n%s", event, logContent)
		}
	}

	// Verify key=value format: iteration_end should have status=ok.
	if !strings.Contains(logContent, "status=ok") {
		t.Errorf("log missing status=ok\nlog content:\n%s", logContent)
	}

	// Verify loop_end has counts.
	if !strings.Contains(logContent, "total=1") {
		t.Errorf("log missing total=1\nlog content:\n%s", logContent)
	}
	if !strings.Contains(logContent, "succeeded=1") {
		t.Errorf("log missing succeeded=1\nlog content:\n%s", logContent)
	}
	if !strings.Contains(logContent, "failed=0") {
		t.Errorf("log missing failed=0\nlog content:\n%s", logContent)
	}
}

func TestLogFormat_MatchesKeyValueSpec(t *testing.T) {
	dir := fakeClaude(t, 0)
	logDir := t.TempDir()

	cmd := exec.Command(afkBin, "-n", "1", "-prompt", "hello", "-log", logDir)
	cmd.Env = append(os.Environ(), "PATH="+dir)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	entries, err := os.ReadDir(logDir)
	if err != nil || len(entries) == 0 {
		t.Fatal("no log file created")
	}

	content, err := os.ReadFile(filepath.Join(logDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	for i, line := range lines {
		// Each line should start with an RFC3339 timestamp, then " [afk] ", then event name.
		if !strings.Contains(line, "[afk]") {
			t.Errorf("line %d missing [afk] tag: %s", i+1, line)
		}

		parts := strings.SplitN(line, " [afk] ", 2)
		if len(parts) != 2 {
			t.Errorf("line %d: cannot split on [afk]: %s", i+1, line)
			continue
		}

		// Verify timestamp parses as RFC3339.
		ts := parts[0]
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			t.Errorf("line %d: timestamp %q is not RFC3339: %v", i+1, ts, err)
		}
	}
}

func TestDaemon_StopsOnSIGINT(t *testing.T) {
	dir := fakeClaude(t, 0)
	cmd := exec.Command(afkBin, "-d", "-prompt", "hello")
	cmd.Env = append(os.Environ(), "PATH="+dir)
	// Need own process group so SIGINT goes only to the child.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Give the daemon time to start running.
	time.Sleep(500 * time.Millisecond)

	// Send SIGINT to the process.
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("daemon exited with error: %v", err)
		}
		if code := cmd.ProcessState.ExitCode(); code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("daemon did not stop within 5 seconds after SIGINT")
	}
}
