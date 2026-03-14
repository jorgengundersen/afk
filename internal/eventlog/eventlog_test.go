package eventlog_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/jorgengundersen/afk/internal/eventlog"
)

func TestF(t *testing.T) {
	f := eventlog.F("key", "value")
	if f.Key != "key" {
		t.Errorf("F().Key = %q, want %q", f.Key, "key")
	}
	if f.Value != "value" {
		t.Errorf("F().Value = %q, want %q", f.Value, "value")
	}
}

func TestNew_creates_log_file(t *testing.T) {
	dir := t.TempDir()
	l, err := eventlog.New(dir, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer l.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in log dir, got %d", len(entries))
	}
	name := entries[0].Name()
	if ext := filepath.Ext(name); ext != ".log" {
		t.Errorf("log file extension = %q, want %q", ext, ".log")
	}
}

func TestNew_creates_missing_directory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "logs")
	l, err := eventlog.New(dir, false)
	if err != nil {
		t.Fatalf("New() should create missing dir, got error: %v", err)
	}
	defer l.Close()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("log dir was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("log dir path is not a directory")
	}
}

func TestNew_returns_error_for_bad_dir(t *testing.T) {
	_, err := eventlog.New("/no/such/dir", false)
	if err == nil {
		t.Fatal("New() with bad dir returned nil error")
	}
}

func readLog(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in log dir, got %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return string(data)
}

// RFC 3339 UTC timestamp pattern
var tsPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)

func TestEvent_output_format(t *testing.T) {
	dir := t.TempDir()
	l, err := eventlog.New(dir, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	l.Event("session-start", eventlog.F("mode", "daemon"), eventlog.F("harness", "claude"))
	l.Close()

	content := readLog(t, dir)
	line := strings.TrimSpace(content)

	if !tsPattern.MatchString(line) {
		t.Errorf("line does not start with RFC3339 timestamp: %q", line)
	}
	if !strings.Contains(line, "[afk]") {
		t.Errorf("line missing [afk] tag: %q", line)
	}
	if !strings.Contains(line, "session-start") {
		t.Errorf("line missing event name: %q", line)
	}
	if !strings.Contains(line, "mode=daemon") {
		t.Errorf("line missing mode=daemon: %q", line)
	}
	if !strings.Contains(line, "harness=claude") {
		t.Errorf("line missing harness=claude: %q", line)
	}
}

func TestEvent_no_fields(t *testing.T) {
	dir := t.TempDir()
	l, err := eventlog.New(dir, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	l.Event("waking")
	l.Close()

	content := readLog(t, dir)
	line := strings.TrimSpace(content)

	if !strings.HasSuffix(line, "waking") {
		t.Errorf("line should end with event name when no fields: %q", line)
	}
}

func TestEvent_quotes_values_with_spaces(t *testing.T) {
	dir := t.TempDir()
	l, err := eventlog.New(dir, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	l.Event("iteration-start", eventlog.F("title", "Fix auth bug"))
	l.Close()

	content := readLog(t, dir)
	line := strings.TrimSpace(content)

	if !strings.Contains(line, `title="Fix auth bug"`) {
		t.Errorf("value with spaces should be quoted: %q", line)
	}
}

func TestEvent_mirrors_to_stderr(t *testing.T) {
	dir := t.TempDir()

	// Capture stderr
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	l, err := eventlog.New(dir, true)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	l.Event("session-start", eventlog.F("mode", "daemon"))
	l.Close()
	w.Close()

	var buf strings.Builder
	data := make([]byte, 4096)
	n, _ := r.Read(data)
	buf.Write(data[:n])
	r.Close()

	stderrOutput := buf.String()
	if !strings.Contains(stderrOutput, "session-start") {
		t.Errorf("stderr should contain event when stderr=true, got %q", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "mode=daemon") {
		t.Errorf("stderr should contain fields, got %q", stderrOutput)
	}

	// Also verify file was written
	fileContent := readLog(t, dir)
	if !strings.Contains(fileContent, "session-start") {
		t.Error("log file should still be written when stderr=true")
	}
}

func TestEvent_no_stderr_when_disabled(t *testing.T) {
	dir := t.TempDir()

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	l, err := eventlog.New(dir, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	l.Event("session-start")
	l.Close()
	w.Close()

	data := make([]byte, 4096)
	n, _ := r.Read(data)
	r.Close()

	if n > 0 {
		t.Errorf("stderr should be empty when stderr=false, got %q", string(data[:n]))
	}
}

func TestClose_flushes_content(t *testing.T) {
	dir := t.TempDir()
	l, err := eventlog.New(dir, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	l.Event("session-start", eventlog.F("mode", "daemon"))
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// After close, file should have content
	content := readLog(t, dir)
	if !strings.Contains(content, "session-start") {
		t.Error("log file should contain event after Close()")
	}

	// Second close should return error (already closed)
	if err := l.Close(); err == nil {
		t.Error("second Close() should return error")
	}
}

func TestEvent_multiple_lines(t *testing.T) {
	dir := t.TempDir()
	l, err := eventlog.New(dir, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	l.Event("session-start", eventlog.F("mode", "daemon"))
	l.Event("iteration-start", eventlog.F("iteration", "1"))
	l.Event("iteration-end", eventlog.F("iteration", "1"), eventlog.F("exit-code", "0"))
	l.Close()

	content := readLog(t, dir)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), content)
	}
}

func TestEvent_concurrent_writes(t *testing.T) {
	dir := t.TempDir()
	l, err := eventlog.New(dir, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(n int) {
			defer wg.Done()
			l.Event("test-event", eventlog.F("goroutine", fmt.Sprintf("%d", n)))
		}(i)
	}
	wg.Wait()
	l.Close()

	content := readLog(t, dir)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != goroutines {
		t.Fatalf("expected %d lines, got %d", goroutines, len(lines))
	}

	// Verify no interleaved lines — each line should match the format
	for i, line := range lines {
		if !tsPattern.MatchString(line) {
			t.Errorf("line %d is malformed (possible interleave): %q", i, line)
		}
	}
}
