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

const itemsCommandStdoutLimitBytes = 8 * 1024 * 1024

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
