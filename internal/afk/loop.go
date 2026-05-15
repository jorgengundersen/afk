package afk

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func RunLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	if cfg.LoopsExplicit {
		return runFixedNonItemLoops(cfg, stdin, stdout, stderr)
	}

	if cfg.Daemon && !cfg.ItemsExplicit && !cfg.ItemsCmdExplicit {
		return runDaemonNonItemLoop(cfg, stdin, stdout, stderr)
	}

	return 0
}

func runFixedNonItemLoops(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	for i := 1; i <= cfg.Loops; i++ {
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, i, cfg.Timeout)
		if err != nil {
			if code, diagnostic, ok := classifyMainChildStartFailure(cfg.CommandArgv[0], err); ok {
				fmt.Fprintln(stderr, diagnostic)
				return code
			}
			fmt.Fprintf(stderr, "execution error: %v\n", err)
			return 1
		}
		if exitCode != 0 && cfg.Fail == "stop" {
			return exitCode
		}
	}

	return 0
}

func runDaemonNonItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	for i := 1; ; i++ {
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, i, cfg.Timeout)
		if err != nil {
			if code, diagnostic, ok := classifyMainChildStartFailure(cfg.CommandArgv[0], err); ok {
				fmt.Fprintln(stderr, diagnostic)
				return code
			}
			fmt.Fprintf(stderr, "execution error: %v\n", err)
			return 1
		}
		if exitCode != 0 && cfg.Fail == "stop" {
			return exitCode
		}
	}
}

func runMainChild(argv []string, stdin io.Reader, stdout, stderr io.Writer, iteration int, timeout time.Duration) (int, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = EnvForNonItemInvocation(os.Environ(), iteration)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	if timeout <= 0 {
		return mapMainChildWaitResult(<-waitCh)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-waitCh:
		return mapMainChildWaitResult(err)
	case <-timer.C:
		if err := signalProcessGroup(cmd.Process.Pid, syscall.SIGTERM); err != nil {
			return 0, fmt.Errorf("timeout cleanup failed: %w", err)
		}

		grace := time.NewTimer(2 * time.Second)
		defer grace.Stop()

		select {
		case <-waitCh:
			return 124, nil
		case <-grace.C:
			if err := signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL); err != nil {
				return 0, fmt.Errorf("timeout cleanup failed: %w", err)
			}
			<-waitCh
			return 124, nil
		}
	}
}

func mapMainChildWaitResult(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	exitErr, ok := err.(*exec.ExitError)
	if ok {
		if waitStatus, ok := exitErr.Sys().(syscall.WaitStatus); ok && waitStatus.Signaled() {
			return 128 + int(waitStatus.Signal()), nil
		}
		return exitErr.ExitCode(), nil
	}

	return 0, err
}

func classifyMainChildStartFailure(command string, err error) (int, string, bool) {
	if errors.Is(err, exec.ErrNotFound) {
		return 127, fmt.Sprintf("command not found: %s", command), true
	}

	if errors.Is(err, os.ErrPermission) {
		return 126, fmt.Sprintf("cannot execute: %s", command), true
	}

	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			switch errno {
			case syscall.EACCES, syscall.ENOEXEC:
				return 126, fmt.Sprintf("cannot execute: %s", command), true
			}
		}
	}

	return 0, "", false
}

func signalProcessGroup(pid int, sig syscall.Signal) error {
	err := syscall.Kill(-pid, sig)
	if err == nil {
		return nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
