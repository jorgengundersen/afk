package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRun_Doctor_RoutesToDoctorPackage(t *testing.T) {
	code := run([]string{"doctor"})
	// doctor.Run returns 0 when healthy; we just verify it doesn't fall
	// through to the normal CLI path (which would return 2 for missing -prompt).
	if code == 2 {
		t.Fatalf("expected doctor subcommand to be routed, got exit code 2 (usage error)")
	}
}

func TestRun_Doctor_ExitCode0(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	code := run([]string{"doctor"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRun_Doctor_JsonFlag(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	code := run([]string{"doctor", "--json"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRun_NoArgs_ExitUsageError(t *testing.T) {
	code := run(nil)
	if code != 2 {
		t.Fatalf("expected exit code 2 (usage error), got %d", code)
	}
}

func TestRun_Help_ExitUsageError(t *testing.T) {
	code := run([]string{"-help"})
	if code != 2 {
		t.Fatalf("expected exit code 2 (usage error), got %d", code)
	}
}

func TestBuild_HelpPrintsUsage(t *testing.T) {
	bin := t.TempDir() + "/afk"
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %s\n%s", err, out)
	}
	cmd := exec.Command(bin, "-help")
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "Usage") && !strings.Contains(string(out), "-prompt") {
		t.Fatalf("expected usage output, got: %s", out)
	}
}

func TestRun_HarnessNotFound_ExitRuntimeError(t *testing.T) {
	// claude binary not in PATH → harness.New fails → exit 1 (runtime error)
	t.Setenv("PATH", t.TempDir())
	code := run([]string{"-prompt", "hello"})
	if code != 1 {
		t.Fatalf("expected exit code 1 (runtime error), got %d", code)
	}
}

// fakeClaude creates a fake "claude" binary that exits with the given code.
func fakeClaude(t *testing.T, exitCode int) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	content := "#!/bin/sh\nexit " + strconv.Itoa(exitCode) + "\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRun_NonZeroHarness_ExitCode0(t *testing.T) {
	// Non-zero harness exit is not a Go error — harness completed, just returned non-zero.
	dir := fakeClaude(t, 1)
	t.Setenv("PATH", dir)
	code := run([]string{"-prompt", "hello", "-n", "1"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRun_Success_ExitCode0(t *testing.T) {
	dir := fakeClaude(t, 0)
	t.Setenv("PATH", dir)
	code := run([]string{"-prompt", "hello", "-n", "1"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}
