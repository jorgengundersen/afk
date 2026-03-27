package harness

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"time"

	"github.com/jorgengundersen/afk/internal/signal"
)

const killGracePeriod = 5 * time.Second

// runCmd executes cmd in its own process group. On context cancellation it
// sends SIGTERM to the entire group, waits a grace period, then escalates
// to SIGKILL. This ensures grandchild processes are cleaned up.
func runCmd(ctx context.Context, cmd *exec.Cmd) (int, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = nil // disable Go's default SIGKILL; we handle cancellation via process group SIGTERM

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	pgid := cmd.Process.Pid

	// Register a force-kill hook so that a second signal (double Ctrl+C)
	// sends SIGKILL to the process group immediately.
	deregister := signal.OnForceKill(func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	})
	defer deregister()

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	select {
	case err := <-waitDone:
		if err == nil {
			return 0, nil
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, err

	case <-ctx.Done():
		_ = syscall.Kill(-pgid, syscall.SIGTERM)

		select {
		case <-waitDone:
		case <-time.After(killGracePeriod):
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
			<-waitDone
		}
		return 0, ctx.Err()
	}
}
