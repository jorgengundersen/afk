package afk

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

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
