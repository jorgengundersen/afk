package harness

import (
	"testing"
)

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
