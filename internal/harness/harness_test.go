package harness

import (
	"context"
	"reflect"
	"testing"
)

// --- Factory tests ---

func TestNewClaudeRunner(t *testing.T) {
	r, err := New("claude", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.(*Claude); !ok {
		t.Fatalf("expected *Claude, got %T", r)
	}
}

func TestNewClaudeWithModel(t *testing.T) {
	r, err := New("claude", "sonnet", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, ok := r.(*Claude)
	if !ok {
		t.Fatalf("expected *Claude, got %T", r)
	}
	if c.model != "sonnet" {
		t.Fatalf("expected model %q, got %q", "sonnet", c.model)
	}
}

func TestNewClaudeWithArgs(t *testing.T) {
	r, err := New("claude", "", "", "--dangerously-skip-permissions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, ok := r.(*Claude)
	if !ok {
		t.Fatalf("expected *Claude, got %T", r)
	}
	if c.harnessArgs != "--dangerously-skip-permissions" {
		t.Fatalf("expected harnessArgs %q, got %q", "--dangerously-skip-permissions", c.harnessArgs)
	}
}

func TestNewOpenCodeRunner(t *testing.T) {
	r, err := New("opencode", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.(*OpenCode); !ok {
		t.Fatalf("expected *OpenCode, got %T", r)
	}
}

func TestNewRawRunner(t *testing.T) {
	r, err := New("claude", "", "my-agent {prompt}", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.(*Raw); !ok {
		t.Fatalf("expected *Raw, got %T", r)
	}
}

func TestNewUnknownHarness(t *testing.T) {
	_, err := New("nope", "", "", "")
	if err == nil {
		t.Fatal("expected error for unknown harness")
	}
	want := `unknown harness "nope"`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestNewRawIgnoresHarnessArgs(t *testing.T) {
	r, err := New("claude", "", "cmd {prompt}", "--foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, ok := r.(*Raw)
	if !ok {
		t.Fatalf("expected *Raw, got %T", r)
	}
	if raw.template != "cmd {prompt}" {
		t.Fatalf("expected template %q, got %q", "cmd {prompt}", raw.template)
	}
}

// --- agentArgs tests (command construction) ---

func TestAgentArgsPromptOnly(t *testing.T) {
	got := agentArgs("do the thing", "", "")
	want := []string{"-p", "do the thing"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestAgentArgsWithModel(t *testing.T) {
	got := agentArgs("do the thing", "sonnet", "")
	want := []string{"-p", "do the thing", "--model", "sonnet"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestAgentArgsWithHarnessArgs(t *testing.T) {
	got := agentArgs("do the thing", "", "--dangerously-skip-permissions --verbose")
	want := []string{"-p", "do the thing", "--dangerously-skip-permissions", "--verbose"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestAgentArgsWithModelAndHarnessArgs(t *testing.T) {
	got := agentArgs("prompt", "sonnet", "--verbose")
	want := []string{"-p", "prompt", "--model", "sonnet", "--verbose"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

// --- runAgent integration tests (subprocess lifecycle) ---

func TestRunAgentExitCode0(t *testing.T) {
	exitCode, err := runAgent(context.Background(), "true", "ignored", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunAgentExitCode1(t *testing.T) {
	exitCode, err := runAgent(context.Background(), "false", "ignored", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunAgentBinaryNotFound(t *testing.T) {
	_, err := runAgent(context.Background(), "nonexistent-binary-xyz", "hello", "", "")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestRunAgentContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := runAgent(ctx, "sleep", "10", "", "")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- Raw.Run tests ---

func TestRawRunSubstitutesPrompt(t *testing.T) {
	r := &Raw{template: "echo {prompt}"}
	exitCode, err := r.Run(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRawRunExitCode1(t *testing.T) {
	r := &Raw{template: "exit 1"}
	exitCode, err := r.Run(context.Background(), "ignored")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestRawRunShellEscapesPrompt(t *testing.T) {
	// Prompt with shell metacharacters should not cause injection
	r := &Raw{template: "echo {prompt}"}
	exitCode, err := r.Run(context.Background(), "hello; exit 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("prompt with semicolon should be escaped, got exit code %d", exitCode)
	}
}

func TestRawRunContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := &Raw{template: "sleep 10"}
	_, err := r.Run(ctx, "ignored")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
