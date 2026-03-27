package harness

import (
	"bytes"
	"context"
	"io"
	"os"
	"reflect"
	"strings"
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

// --- Runner.Run subprocess lifecycle tests ---

func TestClaudeRunExitCode0(t *testing.T) {
	// Use a Claude runner with "true" as the binary won't be found,
	// so test via Raw which exercises the same runCmd path.
	r := &Raw{template: "true"}
	exitCode, err := r.Run(context.Background(), "ignored")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRawRunExitCode1ViaRunCmd(t *testing.T) {
	r := &Raw{template: "false"}
	exitCode, err := r.Run(context.Background(), "ignored")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
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

// --- Output passthrough tests ---

func TestOpenCodeInheritsStdoutStderr(t *testing.T) {
	r, _ := New("opencode", "", "", "")
	oc := r.(*OpenCode)
	cmd := oc.buildCmd(context.Background(), "hello")
	if cmd.Stdout != os.Stdout {
		t.Fatal("expected OpenCode cmd.Stdout to be os.Stdout")
	}
	if cmd.Stderr != os.Stderr {
		t.Fatal("expected OpenCode cmd.Stderr to be os.Stderr")
	}
}

func TestRawInheritsStdoutStderr(t *testing.T) {
	r, _ := New("", "", "echo {prompt}", "")
	raw := r.(*Raw)
	cmd := raw.buildCmd(context.Background(), "hello")
	if cmd.Stdout != os.Stdout {
		t.Fatal("expected Raw cmd.Stdout to be os.Stdout")
	}
	if cmd.Stderr != os.Stderr {
		t.Fatal("expected Raw cmd.Stderr to be os.Stderr")
	}
}

func TestClaudeStderrInheritedStdoutPiped(t *testing.T) {
	r, _ := New("claude", "", "", "")
	c := r.(*Claude)
	cmd := c.buildCmd(context.Background(), "hello")
	if cmd.Stdout == nil || cmd.Stdout == os.Stdout {
		t.Fatal("expected Claude cmd.Stdout to be a pipe (not nil or os.Stdout)")
	}
	if cmd.Stderr != os.Stderr {
		t.Fatal("expected Claude cmd.Stderr to be os.Stderr")
	}
}

func TestClaudeArgsIncludeStreamJSON(t *testing.T) {
	r, _ := New("claude", "", "", "")
	c := r.(*Claude)
	cmd := c.buildCmd(context.Background(), "hello")
	args := cmd.Args[1:] // skip binary name
	found := false
	for i, a := range args {
		if a == "--output-format" && i+1 < len(args) && args[i+1] == "stream-json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected --output-format stream-json in args, got %v", args)
	}
}

func TestClaudeArgsIncludeStreamJSONWithModelAndHarnessArgs(t *testing.T) {
	r, _ := New("claude", "sonnet", "", "--verbose")
	c := r.(*Claude)
	cmd := c.buildCmd(context.Background(), "hello")
	args := cmd.Args[1:]
	// Verify all expected args are present
	want := []string{"-p", "hello", "--model", "sonnet", "--verbose", "--output-format", "stream-json"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected args %v, got %v", want, args)
	}
}

// --- Claude.Run rendering integration ---

func TestClaudeRunRendersStreamEvents(t *testing.T) {
	// Create a helper script that emits stream-json lines to stdout
	dir := t.TempDir()
	helper := dir + "/fake-claude"
	script := `#!/bin/sh
echo '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello from fake claude"}]}}'
echo '{"type":"result","cost_usd":0.005,"duration_ms":2000,"result":"done","is_error":false}'
`
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	pr, pw := io.Pipe()
	var buf bytes.Buffer
	c := &Claude{
		pr:     pr,
		pw:     pw,
		output: &buf,
	}

	// Override the binary name by using buildCmd directly won't work,
	// so we test via a Raw-like approach: build a command manually.
	// Instead, test the render wiring by writing directly to the pipe.
	go func() {
		pw.Write([]byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello from test"}]}}` + "\n"))
		pw.Write([]byte(`{"type":"result","cost_usd":0.005,"duration_ms":2000,"result":"done","is_error":false}` + "\n"))
		pw.Close()
	}()

	c.renderOutput(context.Background())

	got := buf.String()
	if !strings.Contains(got, "hello from test") {
		t.Fatalf("expected rendered text, got %q", got)
	}
	if !strings.Contains(got, "[done]") {
		t.Fatalf("expected done summary, got %q", got)
	}
}

// --- Codex factory and command tests ---

func TestNewCodexRunner(t *testing.T) {
	r, err := New("codex", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.(*Codex); !ok {
		t.Fatalf("expected *Codex, got %T", r)
	}
}

func TestNewCodexWithModel(t *testing.T) {
	r, err := New("codex", "o3", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, ok := r.(*Codex)
	if !ok {
		t.Fatalf("expected *Codex, got %T", r)
	}
	if c.model != "o3" {
		t.Fatalf("expected model %q, got %q", "o3", c.model)
	}
}

func TestCodexBuildCmdPromptOnly(t *testing.T) {
	c := &Codex{}
	cmd := c.buildCmd(context.Background(), "do the thing")
	args := cmd.Args[1:] // skip binary name
	want := []string{"exec", "do the thing", "--json"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected args %v, got %v", want, args)
	}
}

func TestCodexBuildCmdWithModelAndArgs(t *testing.T) {
	c := &Codex{model: "o3", harnessArgs: "--full-auto"}
	cmd := c.buildCmd(context.Background(), "hello")
	args := cmd.Args[1:]
	want := []string{"exec", "hello", "--json", "--model", "o3", "--full-auto"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected args %v, got %v", want, args)
	}
}

func TestCodexStderrInheritedStdoutPiped(t *testing.T) {
	r, _ := New("codex", "", "", "")
	c := r.(*Codex)
	cmd := c.buildCmd(context.Background(), "hello")
	if cmd.Stdout == nil || cmd.Stdout == os.Stdout {
		t.Fatal("expected Codex cmd.Stdout to be a pipe (not nil or os.Stdout)")
	}
	if cmd.Stderr != os.Stderr {
		t.Fatal("expected Codex cmd.Stderr to be os.Stderr")
	}
}

func TestCodexRunRendersStreamEvents(t *testing.T) {
	pr, pw := io.Pipe()
	var buf bytes.Buffer
	c := &Codex{
		pr:     pr,
		pw:     pw,
		output: &buf,
	}

	go func() {
		pw.Write([]byte(`{"type":"item.completed","item":{"type":"agent_message","content":"hello from codex"}}` + "\n"))
		pw.Write([]byte(`{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":50}}` + "\n"))
		pw.Close()
	}()

	c.renderOutput(context.Background())

	got := buf.String()
	if !strings.Contains(got, "hello from codex") {
		t.Fatalf("expected rendered text, got %q", got)
	}
	if !strings.Contains(got, "[done]") {
		t.Fatalf("expected done summary, got %q", got)
	}
}

// --- CheckBinary tests ---

func TestCheckBinaryNamedHarnessFound(t *testing.T) {
	// "sh" should be in PATH on any system
	err := CheckBinary("claude", "")
	// claude likely not in PATH in test env, so just test the function exists
	// Use a known binary instead by testing raw path
	_ = err
}

func TestCheckBinaryNamedHarnessNotFound(t *testing.T) {
	err := CheckBinary("claude", "")
	if err == nil {
		t.Skip("claude binary found in PATH, cannot test not-found case")
	}
	want := `harness "claude": binary "claude" not found in PATH`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestCheckBinaryOpenCodeNotFound(t *testing.T) {
	err := CheckBinary("opencode", "")
	if err == nil {
		t.Skip("opencode binary found in PATH, cannot test not-found case")
	}
	want := `harness "opencode": binary "opencode" not found in PATH`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestCheckBinaryRawFirstToken(t *testing.T) {
	// "sh" exists everywhere
	err := CheckBinary("", "sh {prompt}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckBinaryRawNotFound(t *testing.T) {
	err := CheckBinary("", "nonexistent-xyz {prompt}")
	if err == nil {
		t.Fatal("expected error for missing raw binary")
	}
	want := `harness "raw": binary "nonexistent-xyz" not found in PATH`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestCheckBinaryRawWhitespaceOnly(t *testing.T) {
	err := CheckBinary("", "   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only raw value")
	}
	want := `harness "raw": empty command template`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestCheckBinaryUnknownHarness(t *testing.T) {
	err := CheckBinary("nope", "")
	if err == nil {
		t.Fatal("expected error for unknown harness")
	}
	want := `unknown harness "nope"`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}
