//go:build !unix

package harness

import (
	"context"
	"errors"
	"os/exec"
)

// runCmd executes cmd using the default os/exec behaviour.
// No process group management — context cancellation only kills the direct child.
func runCmd(ctx context.Context, cmd *exec.Cmd) (int, error) {
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}
