package harness

import (
	"context"
	"os"
	"os/exec"

	"github.com/jorgengundersen/afk/internal/config"
)

type openCode struct {
	model      string
	agentFlags string
}

func newOpenCode(cfg config.Config) *openCode {
	return &openCode{model: cfg.Model, agentFlags: cfg.AgentFlags}
}

// Run invokes opencode -p <prompt> --yes [--model M] [extra args].
func (o *openCode) Run(ctx context.Context, prompt string) (int, error) {
	args := []string{"-p", prompt, "--yes"}
	if o.model != "" {
		args = append(args, "--model", o.model)
	}
	if o.agentFlags != "" {
		args = append(args, o.agentFlags)
	}

	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}
