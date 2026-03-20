package doctor_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jorgengundersen/afk/internal/doctor"
)

// fakeBin creates a fake executable in a temp directory and returns the directory.
func fakeBin(t *testing.T, name string) string {
	t.Helper()
	return fakeBinWithOutput(t, name, "", 0)
}

// fakeBinWithOutput creates a fake executable that prints output and exits with code.
func fakeBinWithOutput(t *testing.T, name, output string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n"
	if output != "" {
		// Write output to a data file and cat it with absolute path,
		// so the script works even with a restricted PATH.
		dataPath := filepath.Join(dir, name+".json")
		if err := os.WriteFile(dataPath, []byte(output), 0o644); err != nil {
			t.Fatalf("writing fake output: %v", err)
		}
		script += "/bin/cat " + dataPath + "\n"
	}
	if exitCode != 0 {
		script += "exit " + strconv.Itoa(exitCode) + "\n"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake binary: %v", err)
	}
	return dir
}

func TestCollect_HarnessStatuses(t *testing.T) {
	// Put only "claude" in PATH.
	claudeDir := fakeBin(t, "claude")
	t.Setenv("PATH", claudeDir)

	report := doctor.Collect()

	if report.Harnesses["claude"].Status != "found" {
		t.Errorf("claude status = %q, want %q", report.Harnesses["claude"].Status, "found")
	}
	if report.Harnesses["claude"].Path == "" {
		t.Error("claude path is empty, want non-empty")
	}
	if report.Harnesses["opencode"].Status != "not_found" {
		t.Errorf("opencode status = %q, want %q", report.Harnesses["opencode"].Status, "not_found")
	}
	if report.Harnesses["codex"].Status != "not_found" {
		t.Errorf("codex status = %q, want %q", report.Harnesses["codex"].Status, "not_found")
	}
	if report.Harnesses["copilot"].Status != "not_found" {
		t.Errorf("copilot status = %q, want %q", report.Harnesses["copilot"].Status, "not_found")
	}
}

func TestCollect_BeadsBinaryFound(t *testing.T) {
	bdDir := fakeBin(t, "bd")
	t.Setenv("PATH", bdDir)

	report := doctor.Collect()

	if report.Beads.Binary.Status != "found" {
		t.Errorf("beads binary status = %q, want %q", report.Beads.Binary.Status, "found")
	}
	if report.Beads.Binary.Path == "" {
		t.Error("beads binary path is empty, want non-empty")
	}
}

func TestCollect_BeadsBinaryNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	report := doctor.Collect()

	if report.Beads.Binary.Status != "not_found" {
		t.Errorf("beads binary status = %q, want %q", report.Beads.Binary.Status, "not_found")
	}
}

func TestCollect_BeadsProjectFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".beads"), 0o755); err != nil {
		t.Fatalf("creating .beads dir: %v", err)
	}
	t.Chdir(dir)

	report := doctor.Collect()

	if report.Beads.Project.Status != "found" {
		t.Errorf("beads project status = %q, want %q", report.Beads.Project.Status, "found")
	}
	if report.Beads.Project.Path == "" {
		t.Error("beads project path is empty, want non-empty")
	}
}

func TestCollect_BeadsProjectNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	report := doctor.Collect()

	if report.Beads.Project.Status != "not_found" {
		t.Errorf("beads project status = %q, want %q", report.Beads.Project.Status, "not_found")
	}
}

func TestCollect_BeadsDoctorSuccess(t *testing.T) {
	jsonOut := `{
		"path": "/tmp/project",
		"checks": [
			{"name": "Git Repo", "status": "ok", "message": "all good", "category": "git"}
		],
		"overall_ok": true,
		"cli_version": "0.5.0"
	}`
	bdDir := fakeBinWithOutput(t, "bd", jsonOut, 0)
	t.Setenv("PATH", bdDir)

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".beads"), 0o755); err != nil {
		t.Fatalf("creating .beads dir: %v", err)
	}
	t.Chdir(dir)

	report := doctor.Collect()

	if report.Beads.Doctor.Status != "ok" {
		t.Errorf("beads doctor status = %q, want %q", report.Beads.Doctor.Status, "ok")
	}
	if report.Beads.Doctor.CLIVersion != "0.5.0" {
		t.Errorf("cli_version = %q, want %q", report.Beads.Doctor.CLIVersion, "0.5.0")
	}
	if !report.Beads.Doctor.OverallOK {
		t.Error("overall_ok = false, want true")
	}
	if len(report.Beads.Doctor.Checks) != 1 {
		t.Fatalf("len(checks) = %d, want 1", len(report.Beads.Doctor.Checks))
	}
	if report.Beads.Doctor.Checks[0].Name != "Git Repo" {
		t.Errorf("checks[0].name = %q, want %q", report.Beads.Doctor.Checks[0].Name, "Git Repo")
	}
}

func TestCollect_BeadsDoctorSkippedBdNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	report := doctor.Collect()

	if report.Beads.Doctor.Status != "skipped" {
		t.Errorf("beads doctor status = %q, want %q", report.Beads.Doctor.Status, "skipped")
	}
	if report.Beads.Doctor.Reason == "" {
		t.Error("beads doctor reason is empty, want non-empty")
	}
}

func TestCollect_BeadsDoctorSkippedNoProject(t *testing.T) {
	bdDir := fakeBin(t, "bd")
	t.Setenv("PATH", bdDir)
	t.Chdir(t.TempDir()) // no .beads/ dir

	report := doctor.Collect()

	if report.Beads.Doctor.Status != "skipped" {
		t.Errorf("beads doctor status = %q, want %q", report.Beads.Doctor.Status, "skipped")
	}
	if report.Beads.Doctor.Reason == "" {
		t.Error("beads doctor reason is empty, want non-empty")
	}
}

func TestCollect_BeadsDoctorError(t *testing.T) {
	bdDir := fakeBinWithOutput(t, "bd", "", 1)
	t.Setenv("PATH", bdDir)

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".beads"), 0o755); err != nil {
		t.Fatalf("creating .beads dir: %v", err)
	}
	t.Chdir(dir)

	report := doctor.Collect()

	if report.Beads.Doctor.Status != "error" {
		t.Errorf("beads doctor status = %q, want %q", report.Beads.Doctor.Status, "error")
	}
	if report.Beads.Doctor.Error == "" {
		t.Error("beads doctor error is empty, want non-empty")
	}
}
