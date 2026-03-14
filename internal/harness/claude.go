package harness

import (
	"context"
	"os/exec"

	"github.com/jorgengundersen/afk/internal/config"
)

type claude struct {
	agentFlags string
}

func newClaude(cfg config.Config) *claude {
	return &claude{agentFlags: cfg.AgentFlags}
}

// Run invokes claude -p <prompt> --dangerously-skip-permissions [extra args].
func (c *claude) Run(ctx context.Context, prompt string) (int, error) {
	args := []string{"-p", prompt, "--dangerously-skip-permissions"}
	if c.agentFlags != "" {
		args = append(args, c.agentFlags)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}
