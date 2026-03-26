package logger

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLogSingleEvent(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)
	if err := l.Log("test", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	line := string(data)
	// Match: <RFC3339> [afk] test k=v\n
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z \[afk\] test k=v\n$`)
	if !re.MatchString(line) {
		t.Fatalf("log line %q does not match expected pattern", line)
	}
}

func TestLogMultipleFieldsSorted(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)
	if err := l.Log("e", map[string]any{"b": "2", "a": "1"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	line := string(data)
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z \[afk\] e a=1 b=2\n$`)
	if !re.MatchString(line) {
		t.Fatalf("log line %q does not match expected pattern", line)
	}
}

func TestLogValueWithSpacesQuoted(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)
	if err := l.Log("e", map[string]any{"msg": "hello world"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	line := string(data)
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z \[afk\] e msg="hello world"\n$`)
	if !re.MatchString(line) {
		t.Fatalf("log line %q does not match expected pattern", line)
	}
}

func TestLogNoFields(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)
	if err := l.Log("ping", map[string]any{}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	line := string(data)
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z \[afk\] ping\n$`)
	if !re.MatchString(line) {
		t.Fatalf("log line %q does not match expected pattern", line)
	}
}

func TestLogMultipleEventsAppended(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)
	if err := l.Log("first", map[string]any{"n": "1"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Log("second", map[string]any{"n": "2"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), string(data))
	}
	if !strings.Contains(lines[0], "[afk] first n=1") {
		t.Fatalf("first line %q missing expected content", lines[0])
	}
	if !strings.Contains(lines[1], "[afk] second n=2") {
		t.Fatalf("second line %q missing expected content", lines[1])
	}
}

func TestCloseFlushes(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)
	if err := l.Log("flush-test", map[string]any{"ok": "true"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	if !strings.Contains(string(data), "[afk] flush-test ok=true") {
		t.Fatalf("expected event in file after close, got %q", string(data))
	}
}

func TestLogAnyValueTypes(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)
	if err := l.Log("e", map[string]any{"count": 42, "name": "test"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	line := string(data)
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z \[afk\] e count=42 name=test\n$`)
	if !re.MatchString(line) {
		t.Fatalf("log line %q does not match expected pattern", line)
	}
}

func TestLogAfterCloseIsNoop(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)
	if err := l.Log("before", map[string]any{"n": "1"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Remove the log file so we can detect if Log re-opens it.
	os.Remove(tmpFile)

	// Log after Close should return nil — no re-open, no write.
	if err := l.Log("after-close", map[string]any{"n": "2"}); err != nil {
		t.Fatalf("Log after Close returned error: %v", err)
	}

	// File should NOT be recreated.
	if _, err := os.Stat(tmpFile); err == nil {
		t.Fatalf("expected log file to not exist after close, but it was recreated")
	}
}

func TestDoubleCloseReturnsNil(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)
	if err := l.Log("event", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Fatalf("first Close() error: %v", err)
	}

	// Second Close should return nil, not attempt to close an already-closed fd.
	if err := l.Close(); err != nil {
		t.Fatalf("second Close() returned error: %v", err)
	}
}

func TestFileCreatedOnFirstLogNotNew(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	_ = New(tmpFile)

	// File should NOT exist after New — lazy open means no file until Log.
	if _, err := os.Stat(tmpFile); err == nil {
		t.Fatal("file should not exist after New(), before any Log() call")
	}
}

func TestLogReturnsErrorOnBadPath(t *testing.T) {
	// /dev/null/impossible is never a valid directory.
	l := New("/dev/null/impossible/test.log")
	err := l.Log("test", map[string]any{"k": "v"})
	if err == nil {
		t.Fatal("expected error when directory cannot be created, got nil")
	}
}

func TestLogReturnsErrorOnWriteFailure(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	l := New(tmpFile)

	// First Log succeeds — opens the file.
	if err := l.Log("setup", map[string]any{}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}

	// Close the underlying file to force a write error on next Log.
	l.file.Close()

	err := l.Log("after-fd-closed", map[string]any{})
	if err == nil {
		t.Fatal("expected error when writing to closed fd, got nil")
	}
}

func TestLogCreatesMissingDirectory(t *testing.T) {
	// Use a nested path where the parent directory doesn't exist.
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nested", "deep", "test.log")
	l := New(logPath)
	if err := l.Log("test", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected log file to exist after Log with missing dir: %v", err)
	}
	if !strings.Contains(string(data), "[afk] test k=v") {
		t.Fatalf("log content %q missing expected event", string(data))
	}
}

func TestSessionPathCreatesDir(t *testing.T) {
	// Use a temp dir as the base to avoid polluting the real home dir.
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	path, err := SessionPath()
	if err != nil {
		t.Fatalf("SessionPath() error: %v", err)
	}

	// Path should match pattern: <base>/afk/logs/afk-<timestamp>.log
	re := regexp.MustCompile(`afk/logs/afk-\d{8}-\d{6}\.log$`)
	if !re.MatchString(path) {
		t.Fatalf("path %q does not match expected pattern", path)
	}

	// Directory should have been created.
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("log directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %q to be a directory", dir)
	}
}
