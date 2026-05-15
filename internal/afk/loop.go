package afk

import (
	"bytes"
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

	if cfg.ItemsExplicit {
		return runStaticItemLoop(cfg, stdin, stdout, stderr)
	}

	if cfg.ItemsCmdExplicit {
		if cfg.Daemon {
			return runDaemonDynamicItemLoop(cfg, stdout, stderr)
		}
		return runNonDaemonDynamicItemLoop(cfg, stdout, stderr)
	}

	return 0
}

func runFixedNonItemLoops(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	lastNonZero := 0

	for i := 1; i <= cfg.Loops; i++ {
		env := EnvForNonItemInvocation(os.Environ(), i)
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout)
		if err != nil {
			if code, diagnostic, ok := classifyMainChildStartFailure(cfg.CommandArgv[0], err); ok {
				fmt.Fprintln(stderr, diagnostic)
				return code
			}
			fmt.Fprintf(stderr, "execution error: %v\n", err)
			return 1
		}

		if cfg.UntilSuccess {
			if exitCode == 0 {
				return 0
			}
			lastNonZero = exitCode
			continue
		}

		if exitCode != 0 && cfg.Fail == "stop" {
			return exitCode
		}
	}

	if cfg.UntilSuccess {
		if lastNonZero != 0 {
			return lastNonZero
		}
		return 1
	}

	return 0
}

func runDaemonNonItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	for i := 1; ; i++ {
		env := EnvForNonItemInvocation(os.Environ(), i)
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout)
		if err != nil {
			if code, diagnostic, ok := classifyMainChildStartFailure(cfg.CommandArgv[0], err); ok {
				fmt.Fprintln(stderr, diagnostic)
				return code
			}
			fmt.Fprintf(stderr, "execution error: %v\n", err)
			return 1
		}

		if cfg.UntilSuccess {
			if exitCode == 0 {
				return 0
			}
			continue
		}

		if exitCode != 0 && cfg.Fail == "stop" {
			return exitCode
		}
	}
}

func runStaticItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	items, err := ParseStaticItems(cfg.Items)
	if err != nil {
		fmt.Fprintf(stderr, "item parse error: %v\n", err)
		return 1
	}

	for itemIndex, item := range items {
		env := EnvForItemInvocation(os.Environ(), itemIndex+1, item, itemIndex, len(items))
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout)
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

func runNonDaemonDynamicItemLoop(cfg Config, stdout, stderr io.Writer) int {
	invocationIndex := 1
	remainingEmptySleeps := cfg.EmptySleeps

	for {
		items, err := runItemsCommand(cfg.ItemsCmd, stderr, cfg.Timeout)
		if err != nil {
			fmt.Fprintf(stderr, "item source error: %v\n", err)
			return 1
		}

		if len(items) == 0 {
			if remainingEmptySleeps <= 0 {
				return 0
			}
			remainingEmptySleeps--
			if cfg.Sleep > 0 {
				time.Sleep(cfg.Sleep)
			}
			continue
		}

		for itemIndex, item := range items {
			env := EnvForItemInvocation(os.Environ(), invocationIndex, item, itemIndex, len(items))
			exitCode, err := runMainChild(cfg.CommandArgv, nil, stdout, stderr, env, cfg.Timeout)
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
			invocationIndex++
		}

		return 0
	}
}

func runDaemonDynamicItemLoop(cfg Config, stdout, stderr io.Writer) int {
	invocationIndex := 1

	for {
		items, err := runItemsCommand(cfg.ItemsCmd, stderr, cfg.Timeout)
		if err != nil {
			fmt.Fprintf(stderr, "item source error: %v\n", err)
			return 1
		}

		if len(items) == 0 {
			if cfg.Sleep > 0 {
				time.Sleep(cfg.Sleep)
			}
			continue
		}

		for itemIndex, item := range items {
			env := EnvForItemInvocation(os.Environ(), invocationIndex, item, itemIndex, len(items))
			exitCode, err := runMainChild(cfg.CommandArgv, nil, stdout, stderr, env, cfg.Timeout)
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
			invocationIndex++
		}
	}
}

func runItemsCommand(itemsCmd string, stderr io.Writer, timeout time.Duration) ([]string, error) {
	cmd := exec.Command("/bin/sh", "-c", itemsCmd)
	cmd.Env = baseEnvWithoutAFKContext(os.Environ())
	cmd.Stdin = bytes.NewReader(nil)
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var capturedStdout bytes.Buffer
	cmd.Stdout = &capturedStdout

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	if timeout <= 0 {
		if err := <-waitCh; err != nil {
			return nil, err
		}
		return ParseStaticItems(capturedStdout.String())
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-waitCh:
		if err != nil {
			return nil, err
		}
		return ParseStaticItems(capturedStdout.String())
	case <-timer.C:
		if err := signalProcessGroup(cmd.Process.Pid, syscall.SIGTERM); err != nil {
			return nil, fmt.Errorf("timeout cleanup failed: %w", err)
		}

		grace := time.NewTimer(2 * time.Second)
		defer grace.Stop()

		select {
		case <-waitCh:
			return nil, errors.New("source command timed out")
		case <-grace.C:
			if err := signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL); err != nil {
				return nil, fmt.Errorf("timeout cleanup failed: %w", err)
			}
			<-waitCh
			return nil, errors.New("source command timed out")
		}
	}
}

func runMainChild(argv []string, stdin io.Reader, stdout, stderr io.Writer, env []string, timeout time.Duration) (int, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = env
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
