package harness

import (
	"context"
	"fmt"
	"os"
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

func (c *Claude) buildCmd(ctx context.Context, prompt string) *exec.Cmd {
	args := agentArgs(prompt, c.model, c.harnessArgs)
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	return cmd
}

func (c *Claude) Run(ctx context.Context, prompt string) (int, error) {
	return runCmd(ctx, c.buildCmd(ctx, prompt))
}

// OpenCode invokes the opencode CLI in headless mode.
type OpenCode struct {
	model       string
	harnessArgs string
}

func (o *OpenCode) buildCmd(ctx context.Context, prompt string) *exec.Cmd {
	args := agentArgs(prompt, o.model, o.harnessArgs)
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func (o *OpenCode) Run(ctx context.Context, prompt string) (int, error) {
	return runCmd(ctx, o.buildCmd(ctx, prompt))
}

// Raw executes a user-provided command template via sh -c.
type Raw struct {
	template string
}

func (r *Raw) buildCmd(ctx context.Context, prompt string) *exec.Cmd {
	escaped := "'" + strings.ReplaceAll(prompt, "'", "'\"'\"'") + "'"
	cmdStr := strings.ReplaceAll(r.template, "{prompt}", escaped)
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func (r *Raw) Run(ctx context.Context, prompt string) (int, error) {
	return runCmd(ctx, r.buildCmd(ctx, prompt))
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

// CheckBinary verifies the harness binary exists in PATH.
func CheckBinary(harness, raw string) error {
	if raw != "" {
		fields := strings.Fields(raw)
		if len(fields) == 0 {
			return fmt.Errorf("harness %q: empty command template", "raw")
		}
		bin := fields[0]
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("harness %q: binary %q not found in PATH", "raw", bin)
		}
		return nil
	}
	var bin string
	switch harness {
	case "claude":
		bin = "claude"
	case "opencode":
		bin = "opencode"
	default:
		return fmt.Errorf("unknown harness %q", harness)
	}
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("harness %q: binary %q not found in PATH", harness, bin)
	}
	return nil
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
