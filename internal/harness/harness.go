package harness

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes an agent with the given prompt and returns its exit code.
type Runner interface {
	Run(ctx context.Context, prompt string) (exitCode int, err error)
}

// Claude invokes the claude CLI in headless mode.
type Claude struct {
	model       string
	harnessArgs string
}

func (c *Claude) Run(ctx context.Context, prompt string) (int, error) {
	return runAgent(ctx, "claude", prompt, c.model, c.harnessArgs)
}

// OpenCode invokes the opencode CLI in headless mode.
type OpenCode struct {
	model       string
	harnessArgs string
}

func (o *OpenCode) Run(ctx context.Context, prompt string) (int, error) {
	return runAgent(ctx, "opencode", prompt, o.model, o.harnessArgs)
}

// Raw executes a user-provided command template via sh -c.
type Raw struct {
	template string
}

func (r *Raw) Run(ctx context.Context, prompt string) (int, error) {
	panic("not implemented")
}

// agentArgs builds the argument list for a named harness invocation.
func agentArgs(prompt, model, harnessArgs string) []string {
	args := []string{"-p", prompt}
	if model != "" {
		args = append(args, "--model", model)
	}
	if harnessArgs != "" {
		args = append(args, strings.Fields(harnessArgs)...)
	}
	return args
}

func runAgent(ctx context.Context, binary, prompt, model, harnessArgs string) (int, error) {
	return execCmd(ctx, binary, agentArgs(prompt, model, harnessArgs))
}

func execCmd(ctx context.Context, binary string, args []string) (int, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return 0, err
}

// New returns a Runner for the given config.
func New(harness, model, raw, harnessArgs string) (Runner, error) {
	if raw != "" {
		return &Raw{template: raw}, nil
	}
	switch harness {
	case "claude":
		return &Claude{model: model, harnessArgs: harnessArgs}, nil
	case "opencode":
		return &OpenCode{model: model, harnessArgs: harnessArgs}, nil
	default:
		return nil, fmt.Errorf("unknown harness %q", harness)
	}
}
