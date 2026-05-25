package afk

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var errInterrupted = errors.New("interrupted")

const itemsCommandStdoutLimitBytes = 8 * 1024 * 1024

func RunLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	return runLoopWithInterrupts(cfg, stdin, stdout, stderr, nil)
}

func runLoopWithInterrupts(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	if interrupts == nil {
		sigCh := make(chan os.Signal, 2)
		signal.Notify(sigCh, syscall.SIGINT)
		defer signal.Stop(sigCh)
		interrupts = sigCh
	}

	if cfg.LoopsExplicit {
		return runFixedNonItemLoops(cfg, stdin, stdout, stderr, interrupts)
	}

	if cfg.Daemon && !cfg.ItemsExplicit && !cfg.ItemsCmdExplicit {
		return runDaemonNonItemLoop(cfg, stdin, stdout, stderr, interrupts)
	}

	if cfg.ItemsExplicit {
		return runStaticItemLoop(cfg, stdin, stdout, stderr, interrupts)
	}

	if cfg.ItemsCmdExplicit {
		if cfg.Daemon {
			return runDaemonDynamicItemLoop(cfg, stdout, stderr, interrupts)
		}
		return runNonDaemonDynamicItemLoop(cfg, stdout, stderr, interrupts)
	}

	return 0
}

func runFixedNonItemLoops(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	lastNonZero := 0

	for i := 1; i <= cfg.Loops; i++ {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		env := EnvForNonItemInvocation(os.Environ(), i)
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout, interrupts)
		if err != nil {
			return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
		}

		decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
		if decision.recordLastNonZero {
			lastNonZero = exitCode
		}
		if decision.stop {
			return decision.exitCode
		}
	}

	return finalExitAfterCompletedWork(cfg.UntilSuccess, lastNonZero)
}

func runDaemonNonItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	for i := 1; ; i++ {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		env := EnvForNonItemInvocation(os.Environ(), i)
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout, interrupts)
		if err != nil {
			return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
		}

		decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
		if decision.stop {
			return decision.exitCode
		}
	}
}

func runStaticItemLoop(cfg Config, stdin io.Reader, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	items, err := ParseStaticItems(cfg.Items)
	if err != nil {
		fmt.Fprintf(stderr, "item parse error: %v\n", err)
		return 1
	}

	for itemIndex, item := range items {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		env := EnvForItemInvocation(os.Environ(), itemIndex+1, item, itemIndex, len(items))
		exitCode, err := runMainChild(cfg.CommandArgv, stdin, stdout, stderr, env, cfg.Timeout, interrupts)
		if err != nil {
			return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
		}

		decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
		if decision.stop {
			return decision.exitCode
		}
	}

	return finalExitAfterCompletedWork(cfg.UntilSuccess, 0)
}

func runNonDaemonDynamicItemLoop(cfg Config, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	invocationIndex := 1
	remainingEmptySleeps := cfg.EmptySleeps

	for {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		items, err := runItemsCommand(cfg.ItemsCmd, stderr, cfg.Timeout, interrupts)
		if err != nil {
			if errors.Is(err, errInterrupted) {
				return 130
			}
			fmt.Fprintf(stderr, "item source error: %v\n", err)
			return 1
		}

		if len(items) == 0 {
			if remainingEmptySleeps <= 0 {
				return finalExitAfterCompletedWork(cfg.UntilSuccess, 0)
			}
			remainingEmptySleeps--
			if cfg.Sleep > 0 {
				if interruptibleSleep(cfg.Sleep, interrupts) {
					return interruptExit(stderr)
				}
			}
			continue
		}

		for itemIndex, item := range items {
			if pendingSigint(interrupts) {
				return interruptExit(stderr)
			}

			env := EnvForItemInvocation(os.Environ(), invocationIndex, item, itemIndex, len(items))
			exitCode, err := runMainChild(cfg.CommandArgv, nil, stdout, stderr, env, cfg.Timeout, interrupts)
			if err != nil {
				return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
			}

			decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
			if decision.stop {
				return decision.exitCode
			}
			invocationIndex++
		}

		return finalExitAfterCompletedWork(cfg.UntilSuccess, 0)
	}
}

func runDaemonDynamicItemLoop(cfg Config, stdout, stderr io.Writer, interrupts <-chan os.Signal) int {
	invocationIndex := 1

	for {
		if pendingSigint(interrupts) {
			return interruptExit(stderr)
		}

		items, err := runItemsCommand(cfg.ItemsCmd, stderr, cfg.Timeout, interrupts)
		if err != nil {
			if errors.Is(err, errInterrupted) {
				return 130
			}
			fmt.Fprintf(stderr, "item source error: %v\n", err)
			return 1
		}

		if len(items) == 0 {
			if cfg.Sleep > 0 {
				if interruptibleSleep(cfg.Sleep, interrupts) {
					return interruptExit(stderr)
				}
			}
			continue
		}

		for itemIndex, item := range items {
			if pendingSigint(interrupts) {
				return interruptExit(stderr)
			}

			env := EnvForItemInvocation(os.Environ(), invocationIndex, item, itemIndex, len(items))
			exitCode, err := runMainChild(cfg.CommandArgv, nil, stdout, stderr, env, cfg.Timeout, interrupts)
			if err != nil {
				return mapMainChildExecutionError(cfg.CommandArgv[0], err, stderr)
			}

			decision := decideMainChildExit(cfg.Fail, cfg.UntilSuccess, exitCode)
			if decision.stop {
				return decision.exitCode
			}
			invocationIndex++
		}
	}
}

func runItemsCommand(itemsCmd string, stderr io.Writer, timeout time.Duration, interrupts <-chan os.Signal) ([]string, error) {
	cmd := exec.Command("/bin/sh", "-c", itemsCmd)
	cmd.Env = baseEnvWithoutAFKContext(os.Environ())
	cmd.Stdin = bytes.NewReader(nil)
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = stdoutWrite

	if err := cmd.Start(); err != nil {
		_ = stdoutRead.Close()
		_ = stdoutWrite.Close()
		return nil, err
	}
	_ = stdoutWrite.Close()

	var capturedStdout bytes.Buffer
	copyCh := make(chan error, 1)
	go func() {
		defer stdoutRead.Close()
		copyCh <- copyBoundedItemsCommandStdout(&capturedStdout, stdoutRead)
	}()

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	var timeoutCh <-chan time.Time
	var timer *time.Timer
	if timeout > 0 {
		timer = time.NewTimer(timeout)
		timeoutCh = timer.C
		defer timer.Stop()
	}

	waitDone := false
	stdoutDone := false
	var waitErr error
	interruptState := newInterruptShutdownState()

	for !(waitDone && stdoutDone) {
		select {
		case copyErr := <-copyCh:
			stdoutDone = true
			copyCh = nil
			if copyErr != nil {
				capturedStdout.Reset()
				if !waitDone {
					if err := terminateProcessGroupAfterSourceCaptureFailure(cmd.Process.Pid, waitCh); err != nil {
						return nil, err
					}
					waitDone = true
					waitCh = nil
				}
				return nil, copyErr
			}
		case err := <-waitCh:
			waitDone = true
			waitCh = nil
			waitErr = err
			timeoutCh = nil
		case <-timeoutCh:
			if err := terminateProcessGroupOnTimeout(cmd.Process.Pid, waitCh); err != nil {
				return nil, err
			}
			emitTimeoutDiagnostic(stderr, "--items-cmd", timeout)
			capturedStdout.Reset()
			if copyCh != nil {
				<-copyCh
			}
			return nil, errors.New("source command timed out")
		case sig := <-interrupts:
			decision := interruptState.observe(sig)
			if decision == interruptDecisionIgnore {
				continue
			}
			if !waitDone {
				escalated := gracefulInterruptShutdown(cmd.Process.Pid, waitCh, interrupts, &interruptState)
				emitInterruptDiagnostic(stderr)
				if escalated {
					emitSecondInterruptEscalationDiagnostic(stderr)
				}
			} else {
				emitInterruptDiagnostic(stderr)
			}
			capturedStdout.Reset()
			if copyCh != nil {
				<-copyCh
			}
			return nil, errInterrupted
		}
	}

	if waitErr != nil {
		capturedStdout.Reset()
		return nil, waitErr
	}
	return ParseStaticItems(capturedStdout.String())
}

func copyBoundedItemsCommandStdout(dst *bytes.Buffer, src io.Reader) error {
	buf := make([]byte, 32*1024)
	captured := 0

	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if captured+n > itemsCommandStdoutLimitBytes {
				remaining := itemsCommandStdoutLimitBytes - captured
				if remaining > 0 {
					_, _ = dst.Write(buf[:remaining])
				}
				return fmt.Errorf("--items-cmd stdout limit exceeded (%d bytes)", itemsCommandStdoutLimitBytes)
			}
			_, _ = dst.Write(buf[:n])
			captured += n
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

func runMainChild(argv []string, stdin io.Reader, stdout, stderr io.Writer, env []string, timeout time.Duration, interrupts <-chan os.Signal) (int, error) {
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

	var timeoutCh <-chan time.Time
	var timer *time.Timer
	if timeout > 0 {
		timer = time.NewTimer(timeout)
		timeoutCh = timer.C
		defer timer.Stop()
	}

	interruptState := newInterruptShutdownState()

	for {
		select {
		case err := <-waitCh:
			return mapMainChildWaitResult(err)
		case <-timeoutCh:
			if err := terminateProcessGroupOnTimeout(cmd.Process.Pid, waitCh); err != nil {
				return 0, err
			}
			emitTimeoutDiagnostic(stderr, "main child", timeout)
			return 124, nil
		case sig := <-interrupts:
			decision := interruptState.observe(sig)
			if decision == interruptDecisionIgnore {
				continue
			}
			escalated := gracefulInterruptShutdown(cmd.Process.Pid, waitCh, interrupts, &interruptState)
			emitInterruptDiagnostic(stderr)
			if escalated {
				emitSecondInterruptEscalationDiagnostic(stderr)
			}
			return 130, errInterrupted
		}
	}
}

func mapMainChildExecutionError(command string, err error, stderr io.Writer) int {
	if errors.Is(err, errInterrupted) {
		return 130
	}
	if code, diagnostic, ok := classifyMainChildStartFailure(command, err); ok {
		fmt.Fprintln(stderr, diagnostic)
		return code
	}
	fmt.Fprintf(stderr, "execution error: %v\n", err)
	return 1
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

func terminateProcessGroupOnTimeout(pid int, waitCh <-chan error) error {
	return terminateProcessGroup(pid, waitCh, "timeout cleanup failed")
}

func terminateProcessGroupAfterSourceCaptureFailure(pid int, waitCh <-chan error) error {
	return terminateProcessGroup(pid, waitCh, "item source cleanup failed")
}

func terminateProcessGroup(pid int, waitCh <-chan error, failurePrefix string) error {
	if err := signalProcessGroup(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("%s: %w", failurePrefix, err)
	}

	grace := time.NewTimer(2 * time.Second)
	defer grace.Stop()

	select {
	case <-waitCh:
		return nil
	case <-grace.C:
		if err := signalProcessGroup(pid, syscall.SIGKILL); err != nil {
			return fmt.Errorf("%s: %w", failurePrefix, err)
		}
		<-waitCh
		return nil
	}
}

func gracefulInterruptShutdown(pid int, waitCh <-chan error, interrupts <-chan os.Signal, interruptState *interruptShutdownState) bool {
	_ = signalProcessGroup(pid, syscall.SIGINT)

	grace := time.NewTimer(5 * time.Second)
	defer grace.Stop()

	select {
	case <-waitCh:
		return false
	case sig := <-interrupts:
		if interruptState.observe(sig) == interruptDecisionForceHardShutdown {
			hardInterruptShutdown(pid, waitCh)
			return true
		}
	case <-grace.C:
	}

	_ = signalProcessGroup(pid, syscall.SIGTERM)
	termGrace := time.NewTimer(2 * time.Second)
	defer termGrace.Stop()

	select {
	case <-waitCh:
		return false
	case sig := <-interrupts:
		if interruptState.observe(sig) == interruptDecisionForceHardShutdown {
			hardInterruptShutdown(pid, waitCh)
			return true
		}
	case <-termGrace.C:
	}

	_ = signalProcessGroup(pid, syscall.SIGKILL)
	<-waitCh
	return false
}

func hardInterruptShutdown(pid int, waitCh <-chan error) {
	_ = signalProcessGroup(pid, syscall.SIGKILL)
	waitForChildWithDeadline(waitCh, 500*time.Millisecond)
}

func waitForChildWithDeadline(waitCh <-chan error, maxWait time.Duration) {
	timer := time.NewTimer(maxWait)
	defer timer.Stop()

	select {
	case <-waitCh:
	case <-timer.C:
	}
}

func pendingSigint(interrupts <-chan os.Signal) bool {
	if interrupts == nil {
		return false
	}

	interruptState := newInterruptShutdownState()

	select {
	case sig := <-interrupts:
		return interruptState.observe(sig) == interruptDecisionStartGracefulShutdown
	default:
		return false
	}
}

func isSigint(sig os.Signal) bool {
	if sig == nil {
		return false
	}
	return sig == os.Interrupt || sig == syscall.SIGINT
}

func interruptExit(stderr io.Writer) int {
	emitInterruptDiagnostic(stderr)
	return 130
}

func emitTimeoutDiagnostic(stderr io.Writer, commandRole string, timeout time.Duration) {
	if stderr == nil {
		return
	}
	fmt.Fprintf(stderr, "timeout: %s exceeded %s\n", commandRole, timeout)
}

func emitInterruptDiagnostic(stderr io.Writer) {
	if stderr == nil {
		return
	}
	fmt.Fprintln(stderr, "interrupt: user requested shutdown; exiting 130")
}

func emitSecondInterruptEscalationDiagnostic(stderr io.Writer) {
	if stderr == nil {
		return
	}
	fmt.Fprintln(stderr, "interrupt: second SIGINT received; forcing hard shutdown")
}
