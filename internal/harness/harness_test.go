package harness_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jorgengundersen/afk/internal/config"
	"github.com/jorgengundersen/afk/internal/harness"
)

func TestNew_unknown_harness(t *testing.T) {
	cfg := config.Config{Harness: "unknown"}
	_, err := harness.New(cfg)
	if err == nil {
		t.Fatal("expected error for unknown harness, got nil")
	}
}

func TestNew_claude(t *testing.T) {
	cfg := config.Config{Harness: "claude"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil harness")
	}
}

// fakeBin creates a fake executable script in a temp dir and returns the dir.
// The caller should prepend this dir to PATH.
func fakeBin(t *testing.T, name, script string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake bin: %v", err)
	}
	return dir
}

func TestClaude_Run_exit_zero(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\nexit 0\n"
	binDir := fakeBin(t, "claude", script)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.Config{Harness: "claude"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	exitCode, err := h.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading args file: %v", err)
	}
	want := "-p\ntest prompt\n--dangerously-skip-permissions\n"
	if string(got) != want {
		t.Errorf("args = %q, want %q", string(got), want)
	}
}

func TestClaude_Run_nonzero_exit(t *testing.T) {
	binDir := fakeBin(t, "claude", "#!/bin/sh\nexit 42\n")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.Config{Harness: "claude"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	exitCode, err := h.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("exit code = %d, want 42", exitCode)
	}
}

func TestClaude_Run_context_cancellation(t *testing.T) {
	binDir := fakeBin(t, "claude", "#!/bin/sh\nsleep 60\n")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.Config{Harness: "claude"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = h.Run(ctx, "test prompt")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestClaude_Run_with_agent_flags(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\nexit 0\n"
	binDir := fakeBin(t, "claude", script)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.Config{Harness: "claude", AgentFlags: "--model opus"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = h.Run(context.Background(), "do stuff")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading args file: %v", err)
	}
	want := "-p\ndo stuff\n--dangerously-skip-permissions\n--model opus\n"
	if string(got) != want {
		t.Errorf("args = %q, want %q", string(got), want)
	}
}

func TestNew_opencode(t *testing.T) {
	binDir := fakeBin(t, "opencode", "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.Config{Harness: "opencode"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil harness")
	}
}

func TestOpenCode_Run_exit_zero(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\nexit 0\n"
	binDir := fakeBin(t, "opencode", script)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.Config{Harness: "opencode"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	exitCode, err := h.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading args file: %v", err)
	}
	want := "-p\ntest prompt\n--yes\n"
	if string(got) != want {
		t.Errorf("args = %q, want %q", string(got), want)
	}
}

func TestOpenCode_Run_nonzero_exit(t *testing.T) {
	binDir := fakeBin(t, "opencode", "#!/bin/sh\nexit 42\n")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.Config{Harness: "opencode"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	exitCode, err := h.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("exit code = %d, want 42", exitCode)
	}
}

func TestOpenCode_Run_context_cancellation(t *testing.T) {
	binDir := fakeBin(t, "opencode", "#!/bin/sh\nsleep 60\n")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.Config{Harness: "opencode"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = h.Run(ctx, "test prompt")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestOpenCode_Run_with_agent_flags(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\nexit 0\n"
	binDir := fakeBin(t, "opencode", script)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.Config{Harness: "opencode", AgentFlags: "--model gpt-4"}
	h, err := harness.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = h.Run(context.Background(), "do stuff")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading args file: %v", err)
	}
	want := "-p\ndo stuff\n--yes\n--model gpt-4\n"
	if string(got) != want {
		t.Errorf("args = %q, want %q", string(got), want)
	}
}

func TestNew_binary_not_in_path(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir, no claude binary

	cfg := config.Config{Harness: "claude"}
	_, err := harness.New(cfg)
	if err == nil {
		t.Fatal("expected error when binary not in PATH, got nil")
	}
}
