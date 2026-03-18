package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBdDoctorResult_UnmarshalJSON(t *testing.T) {
	raw := `{
		"path": "/home/user/project",
		"checks": [
			{
				"name": "Git Repository",
				"status": "ok",
				"message": "valid git repo",
				"detail": "branch: main",
				"fix": "",
				"category": "git"
			},
			{
				"name": "Database",
				"status": "warning",
				"message": "database needs vacuum",
				"category": "maintenance"
			}
		],
		"overall_ok": true,
		"cli_version": "0.5.2",
		"timestamp": "2026-03-18T10:00:00Z",
		"platform": {"os": "linux", "arch": "amd64"},
		"suppressed_count": 3
	}`

	var result BdDoctorResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result.Path != "/home/user/project" {
		t.Errorf("Path = %q, want %q", result.Path, "/home/user/project")
	}
	if result.OverallOK != true {
		t.Error("OverallOK = false, want true")
	}
	if result.CLIVersion != "0.5.2" {
		t.Errorf("CLIVersion = %q, want %q", result.CLIVersion, "0.5.2")
	}
	if result.Timestamp != "2026-03-18T10:00:00Z" {
		t.Errorf("Timestamp = %q, want %q", result.Timestamp, "2026-03-18T10:00:00Z")
	}
	if result.Platform["os"] != "linux" {
		t.Errorf("Platform[os] = %q, want %q", result.Platform["os"], "linux")
	}
	if result.SuppressedCount != 3 {
		t.Errorf("SuppressedCount = %d, want 3", result.SuppressedCount)
	}

	if len(result.Checks) != 2 {
		t.Fatalf("len(Checks) = %d, want 2", len(result.Checks))
	}

	check := result.Checks[0]
	if check.Name != "Git Repository" {
		t.Errorf("Checks[0].Name = %q, want %q", check.Name, "Git Repository")
	}
	if check.Status != "ok" {
		t.Errorf("Checks[0].Status = %q, want %q", check.Status, "ok")
	}
	if check.Message != "valid git repo" {
		t.Errorf("Checks[0].Message = %q, want %q", check.Message, "valid git repo")
	}
	if check.Detail != "branch: main" {
		t.Errorf("Checks[0].Detail = %q, want %q", check.Detail, "branch: main")
	}
	if check.Category != "git" {
		t.Errorf("Checks[0].Category = %q, want %q", check.Category, "git")
	}

	check1 := result.Checks[1]
	if check1.Status != "warning" {
		t.Errorf("Checks[1].Status = %q, want %q", check1.Status, "warning")
	}
	if check1.Detail != "" {
		t.Errorf("Checks[1].Detail = %q, want empty", check1.Detail)
	}
}

func TestBdDoctorResult_MinimalJSON(t *testing.T) {
	raw := `{
		"path": "/tmp",
		"checks": [],
		"overall_ok": false,
		"cli_version": "0.1.0"
	}`

	var result BdDoctorResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result.OverallOK != false {
		t.Error("OverallOK = true, want false")
	}
	if result.Timestamp != "" {
		t.Errorf("Timestamp = %q, want empty", result.Timestamp)
	}
	if result.Platform != nil {
		t.Errorf("Platform = %v, want nil", result.Platform)
	}
	if result.SuppressedCount != 0 {
		t.Errorf("SuppressedCount = %d, want 0", result.SuppressedCount)
	}
}

// fakeBd creates a fake bd script that prints output and exits with code.
func fakeBd(t *testing.T, output string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n"
	if output != "" {
		script += "cat <<'FAKEJSON'\n" + output + "\nFAKEJSON\n"
	}
	if exitCode != 0 {
		script += "exit " + string(rune('0'+exitCode)) + "\n"
	}
	path := filepath.Join(dir, "bd")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake bd: %v", err)
	}
	return dir
}

func TestRunBdDoctor_Success(t *testing.T) {
	jsonOut := `{
		"path": "/tmp/project",
		"checks": [
			{"name": "Git Repo", "status": "ok", "message": "all good", "category": "git"}
		],
		"overall_ok": true,
		"cli_version": "0.5.0"
	}`
	binDir := fakeBd(t, jsonOut, 0)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	result, err := runBdDoctor()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CLIVersion != "0.5.0" {
		t.Errorf("CLIVersion = %q, want %q", result.CLIVersion, "0.5.0")
	}
	if !result.OverallOK {
		t.Error("OverallOK = false, want true")
	}
	if len(result.Checks) != 1 {
		t.Fatalf("len(Checks) = %d, want 1", len(result.Checks))
	}
	if result.Checks[0].Name != "Git Repo" {
		t.Errorf("Checks[0].Name = %q, want %q", result.Checks[0].Name, "Git Repo")
	}
}

func TestRunBdDoctor_NonZeroExit(t *testing.T) {
	binDir := fakeBd(t, "", 1)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	_, err := runBdDoctor()
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}
}

func TestRunBdDoctor_InvalidJSON(t *testing.T) {
	binDir := fakeBd(t, "not json at all", 0)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	_, err := runBdDoctor()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestRunBdDoctor_BdNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := runBdDoctor()
	if err == nil {
		t.Fatal("expected error when bd not in PATH, got nil")
	}
}
