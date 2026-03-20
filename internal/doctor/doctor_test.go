package doctor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestRun_ReturnsZero(t *testing.T) {
	var buf bytes.Buffer
	code := Run(&buf, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestPrintJSON_MarshalReport(t *testing.T) {
	r := Report{
		Harnesses: map[string]HarnessStatus{
			"claude": {Status: "found", Path: "/usr/bin/claude"},
		},
	}
	r.Beads.Binary = BeadsBinaryStatus{Status: "found", Path: "/usr/bin/bd"}
	r.Beads.Project = BeadsProjectStatus{Status: "found", Path: ".beads"}
	r.Beads.Doctor = BeadsDoctorStatus{Status: "ok", OverallOK: true, CLIVersion: "0.5.0"}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, r); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}

	// Output must be valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}

	// Spot-check a few fields.
	harnesses, ok := parsed["harnesses"].(map[string]any)
	if !ok {
		t.Fatal("missing harnesses key")
	}
	claude, ok := harnesses["claude"].(map[string]any)
	if !ok {
		t.Fatal("missing claude key")
	}
	if claude["status"] != "found" {
		t.Errorf("claude status = %v, want found", claude["status"])
	}

	// Must be indented (2-space).
	if !bytes.Contains(buf.Bytes(), []byte("  ")) {
		t.Error("expected indented JSON output")
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("write failed")
}

func TestPrintJSON_WriteError(t *testing.T) {
	r := Report{Harnesses: map[string]HarnessStatus{}}
	err := PrintJSON(errWriter{}, r)
	if err == nil {
		t.Fatal("expected error from failing writer, got nil")
	}
}

func TestPrintHuman_FullReport(t *testing.T) {
	r := Report{
		Harnesses: map[string]HarnessStatus{
			"claude":   {Status: "found", Path: "/usr/local/bin/claude"},
			"opencode": {Status: "not_found"},
			"codex":    {Status: "not_found"},
			"copilot":  {Status: "not_found"},
		},
	}
	r.Beads.Binary = BeadsBinaryStatus{Status: "found", Path: "/usr/local/bin/bd"}
	r.Beads.Project = BeadsProjectStatus{Status: "found", Path: ".beads"}
	r.Beads.Doctor = BeadsDoctorStatus{
		Status:     "ok",
		OverallOK:  true,
		CLIVersion: "0.5.0",
		Checks: []BdDoctorCheck{
			{Name: "Git Repo", Status: "ok", Message: "all good", Category: "git"},
			{Name: "Git Hooks", Status: "warning", Message: "hook not installed", Category: "git"},
		},
	}

	var buf bytes.Buffer
	PrintHuman(&buf, r)
	out := buf.String()

	// Must contain header.
	assertContains(t, out, "afk doctor")

	// Must contain harness section.
	assertContains(t, out, "Harnesses")
	assertContains(t, out, "claude")
	assertContains(t, out, "found (/usr/local/bin/claude)")
	assertContains(t, out, "not found")

	// Must contain beads section.
	assertContains(t, out, "Beads")
	assertContains(t, out, "bd binary")
	assertContains(t, out, "found (/usr/local/bin/bd)")
	assertContains(t, out, "project (.beads/)")
	assertContains(t, out, "health")
	assertContains(t, out, "ok (v0.5.0, 2 checks passed)")

	// Must contain warnings section.
	assertContains(t, out, "warnings:")
	assertContains(t, out, "Git Hooks: hook not installed (category: git)")

	// Must contain errors section showing none.
	assertContains(t, out, "errors:")
	assertContains(t, out, "(none)")
}

func TestPrintHuman_SkippedDoctor(t *testing.T) {
	r := Report{
		Harnesses: map[string]HarnessStatus{
			"claude":   {Status: "not_found"},
			"opencode": {Status: "not_found"},
			"codex":    {Status: "not_found"},
			"copilot":  {Status: "not_found"},
		},
	}
	r.Beads.Binary = BeadsBinaryStatus{Status: "not_found"}
	r.Beads.Project = BeadsProjectStatus{Status: "not_found"}
	r.Beads.Doctor = BeadsDoctorStatus{Status: "skipped", Reason: "bd not found"}

	var buf bytes.Buffer
	PrintHuman(&buf, r)
	out := buf.String()

	assertContains(t, out, "health")
	assertContains(t, out, "skipped")

	// Warnings and errors sections should be omitted.
	if bytes.Contains([]byte(out), []byte("warnings:")) {
		t.Error("expected warnings section to be omitted when doctor is skipped")
	}
	if bytes.Contains([]byte(out), []byte("errors:")) {
		t.Error("expected errors section to be omitted when doctor is skipped")
	}
}

func TestRun_DefaultHumanOutput(t *testing.T) {
	// Run without --json should produce human-readable output.
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	var buf bytes.Buffer
	Run(&buf, nil)
	out := buf.String()

	assertContains(t, out, "afk doctor")
	assertContains(t, out, "Harnesses")
}

func TestRun_JSONFlag(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	var buf bytes.Buffer
	Run(&buf, []string{"--json"})
	out := buf.String()

	// Must be valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if _, ok := parsed["harnesses"]; !ok {
		t.Error("missing harnesses key in JSON output")
	}
}

func TestRun_ExitCodeOneOnError(t *testing.T) {
	// When bd doctor reports a check with error status, exit code should be 1.
	// We need to set up the report to have an error check.
	// Since Run calls Collect internally, we use a fake bd that returns an error check.
	bdJSON := `{
		"path": "/tmp/project",
		"checks": [
			{"name": "Schema", "status": "error", "message": "schema mismatch", "category": "core"}
		],
		"overall_ok": false,
		"cli_version": "0.5.0"
	}`
	bdDir := fakeBinWithOutput(t, "bd", bdJSON, 0)
	t.Setenv("PATH", bdDir)

	dir := t.TempDir()
	if err := os.Mkdir(dir+"/.beads", 0o755); err != nil {
		t.Fatalf("creating .beads dir: %v", err)
	}
	t.Chdir(dir)

	var buf bytes.Buffer
	code := Run(&buf, nil)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestRun_ExitCodeZeroWhenHealthy(t *testing.T) {
	bdJSON := `{
		"path": "/tmp/project",
		"checks": [
			{"name": "Git Repo", "status": "ok", "message": "all good", "category": "git"}
		],
		"overall_ok": true,
		"cli_version": "0.5.0"
	}`
	bdDir := fakeBinWithOutput(t, "bd", bdJSON, 0)
	t.Setenv("PATH", bdDir)

	dir := t.TempDir()
	if err := os.Mkdir(dir+"/.beads", 0o755); err != nil {
		t.Fatalf("creating .beads dir: %v", err)
	}
	t.Chdir(dir)

	var buf bytes.Buffer
	code := Run(&buf, nil)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

// fakeBinWithOutput creates a fake executable that prints output and exits with code.
func fakeBinWithOutput(t *testing.T, name, output string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n"
	if output != "" {
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

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !bytes.Contains([]byte(s), []byte(substr)) {
		t.Errorf("output missing %q\ngot:\n%s", substr, s)
	}
}
