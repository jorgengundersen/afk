package beads_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jorgengundersen/afk/beads"
)

// fakeBd creates a fake bd executable that outputs the given JSON and exits with code.
func fakeBd(t *testing.T, output string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n"
	if output != "" {
		script += "cat <<'FAKEJSON'\n" + output + "\nFAKEJSON\n"
	}
	script += "exit " + itoa(exitCode) + "\n"
	path := filepath.Join(dir, "bd")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake bd: %v", err)
	}
	return dir
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return string(rune('0' + n))
}

func TestNewClient(t *testing.T) {
	c := beads.NewClient([]string{"bug", "feature"})
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestReady_happy_path(t *testing.T) {
	jsonOut := `[{"id":"test-1","title":"Fix bug","description":"Fix it","status":"open"}]`
	binDir := fakeBd(t, jsonOut, 0)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	c := beads.NewClient(nil)
	issues, err := c.Ready(context.Background())
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].ID != "test-1" {
		t.Errorf("ID = %q, want %q", issues[0].ID, "test-1")
	}
	if issues[0].Title != "Fix bug" {
		t.Errorf("Title = %q, want %q", issues[0].Title, "Fix bug")
	}
	if issues[0].Raw == nil {
		t.Error("Raw should not be nil")
	}
}

func TestReady_no_work(t *testing.T) {
	binDir := fakeBd(t, "[]", 0)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	c := beads.NewClient(nil)
	_, err := c.Ready(context.Background())
	if !errors.Is(err, beads.ErrNoWork) {
		t.Fatalf("got err=%v, want ErrNoWork", err)
	}
}

func TestReady_bd_not_found(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir, no bd binary

	c := beads.NewClient(nil)
	_, err := c.Ready(context.Background())
	if !errors.Is(err, beads.ErrBdNotFound) {
		t.Fatalf("got err=%v, want ErrBdNotFound", err)
	}
}

func TestReady_malformed_json(t *testing.T) {
	binDir := fakeBd(t, "not valid json{{{", 0)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	c := beads.NewClient(nil)
	_, err := c.Ready(context.Background())
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if errors.Is(err, beads.ErrNoWork) || errors.Is(err, beads.ErrBdNotFound) {
		t.Fatalf("got sentinel %v, want generic JSON error", err)
	}
}

func TestReady_bd_exit_nonzero(t *testing.T) {
	binDir := fakeBd(t, "", 1)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	c := beads.NewClient(nil)
	_, err := c.Ready(context.Background())
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}
}

func TestReady_context_cancellation(t *testing.T) {
	// bd that sleeps forever
	dir := t.TempDir()
	script := "#!/bin/sh\nsleep 60\n"
	path := filepath.Join(dir, "bd")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake bd: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := beads.NewClient(nil)
	_, err := c.Ready(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestReady_with_labels(t *testing.T) {
	// Fake bd that echoes its args to a file so we can inspect them
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\necho '[]'\nexit 0\n"
	binDir := t.TempDir()
	path := filepath.Join(binDir, "bd")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake bd: %v", err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	c := beads.NewClient([]string{"bug", "urgent"})
	_, _ = c.Ready(context.Background()) // will return ErrNoWork, that's fine

	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading args file: %v", err)
	}
	want := "ready\n--json\n--label\nbug\n--label\nurgent\n"
	if string(got) != want {
		t.Errorf("args = %q, want %q", string(got), want)
	}
}
