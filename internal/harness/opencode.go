package harness

import (
	"context"
	"os/exec"

	"github.com/jorgengundersen/afk/internal/config"
)

type openCode struct {
	agentFlags string
}

func newOpenCode(cfg config.Config) *openCode {
	return &openCode{agentFlags: cfg.AgentFlags}
}

// Run invokes opencode -p <prompt> --yes [extra args].
func (o *openCode) Run(ctx context.Context, prompt string) (int, error) {
	args := []string{"-p", prompt, "--yes"}
	if o.agentFlags != "" {
		args = append(args, o.agentFlags)
	}

	cmd := exec.CommandContext(ctx, "opencode", args...)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}
