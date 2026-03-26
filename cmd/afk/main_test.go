package main

import (
	"os/exec"
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

	prompt, id, title, ok := ws.Next()
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

	_, _, _, ok := ws.Next()
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
}

func (f *fakeBeadsClient) Ready() ([]issueResult, error) {
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
