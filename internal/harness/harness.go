package harness

import (
	"context"
	"fmt"
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
	panic("not implemented")
}

// OpenCode invokes the opencode CLI in headless mode.
type OpenCode struct {
	model       string
	harnessArgs string
}

func (o *OpenCode) Run(ctx context.Context, prompt string) (int, error) {
	panic("not implemented")
}

// Raw executes a user-provided command template via sh -c.
type Raw struct {
	template string
}

func (r *Raw) Run(ctx context.Context, prompt string) (int, error) {
	panic("not implemented")
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
