package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func build(t *testing.T) string {
	t.Helper()
	bin := t.TempDir() + "/afk"
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}
	return bin
}

func TestMissingPFlag(t *testing.T) {
	bin := build(t)

	cmd := exec.Command(bin)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()

	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got %v", err)
	}

	if !strings.Contains(stderr.String(), "no prompt") {
		t.Errorf("stderr = %q, want it to mention missing prompt", stderr.String())
	}
}

func TestRunRawHarness(t *testing.T) {
	bin := build(t)

	cmd := exec.Command(bin, "--raw", "echo {prompt}", "-p", "hello world", "-n", "1")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}

	got := strings.TrimRight(string(out), "\n")
	if got != "hello world" {
		t.Errorf("stdout = %q, want %q", got, "hello world")
	}
}

func TestBeadsWorkSourceReturnsWork(t *testing.T) {
	ws := &beadsWorkSource{
		client: &fakeBeadsClient{
			issues: []fakeIssue{
				{id: "afk-1", title: "Fix bug", rawJSON: `{"id":"afk-1","title":"Fix bug"}`},
			},
		},
		userPrompt: "focus on tests",
	}

	prompt, id, title, ok, err := ws.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected work to be available")
	}
	if id != "afk-1" {
		t.Errorf("issueID = %q, want %q", id, "afk-1")
	}
	if title != "Fix bug" {
		t.Errorf("issueTitle = %q, want %q", title, "Fix bug")
	}
	if prompt == "" {
		t.Error("prompt should not be empty")
	}
}

func TestBeadsWorkSourceNoWork(t *testing.T) {
	ws := &beadsWorkSource{
		client:     &fakeBeadsClient{issues: nil},
		userPrompt: "",
	}

	_, _, _, ok, err := ws.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected no work available")
	}
}

type fakeIssue struct {
	id      string
	title   string
	rawJSON string
}

type fakeBeadsClient struct {
	issues []fakeIssue
	err    error
}

func (f *fakeBeadsClient) Ready() ([]issueResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	var results []issueResult
	for _, i := range f.issues {
		results = append(results, issueResult{
			ID:      i.id,
			Title:   i.title,
			RawJSON: []byte(i.rawJSON),
		})
	}
	return results, nil
}

func TestBeadsWorkSourceReadyError(t *testing.T) {
	ws := &beadsWorkSource{
		client: &fakeBeadsClient{
			err: errors.New("bd: connection refused"),
		},
		userPrompt: "",
	}

	_, _, _, ok, err := ws.Next()
	if ok {
		t.Error("expected no work available on error")
	}
	if err == nil {
		t.Fatal("expected error to be propagated")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("expected error to contain 'connection refused', got %q", err.Error())
	}
}

// fakeBD creates a shell script that mimics "bd ready --json" output and
// returns the directory containing it (to prepend to PATH).
func fakeBD(t *testing.T, jsonOutput string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "bd")
	content := "#!/bin/sh\necho '" + jsonOutput + "'\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRunBeadsHappyPath(t *testing.T) {
	bin := build(t)

	issueJSON := `[{"id":"afk-42","title":"Fix tests","description":"broken","status":"open","priority":1,"issue_type":"bug"}]`
	bdDir := fakeBD(t, issueJSON)

	cmd := exec.Command(bin, "--beads", "--raw", "echo {prompt}", "-n", "1")
	cmd.Env = append(os.Environ(), "PATH="+bdDir+":"+os.Getenv("PATH"))
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("expected exit 0, got exit %d\nstderr: %s", exitErr.ExitCode(), exitErr.Stderr)
		}
		t.Fatalf("expected exit 0, got error: %v", err)
	}

	got := string(out)

	// Verify the prompt contains the issue instruction
	if !strings.Contains(got, "You have been assigned the following issue") {
		t.Errorf("output should contain issue instruction, got %q", got)
	}

	// Verify the prompt contains the issue JSON
	if !strings.Contains(got, "afk-42") {
		t.Errorf("output should contain issue ID, got %q", got)
	}
	if !strings.Contains(got, "Fix tests") {
		t.Errorf("output should contain issue title, got %q", got)
	}
}

func TestRunBeadsNoWork(t *testing.T) {
	bin := build(t)

	bdDir := fakeBD(t, "[]")

	cmd := exec.Command(bin, "--beads", "--raw", "echo {prompt}", "-n", "1")
	cmd.Env = append(os.Environ(), "PATH="+bdDir+":"+os.Getenv("PATH"))
	err := cmd.Run()

	// When there's no work, the loop exits cleanly with code 0
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}
}

func TestHarnessBinaryNotFound(t *testing.T) {
	bin := build(t)

	// Default harness is "claude" which won't be in PATH during tests
	cmd := exec.Command(bin, "-p", "hello", "-n", "1")
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + t.TempDir()}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()

	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got %v", err)
	}
}
