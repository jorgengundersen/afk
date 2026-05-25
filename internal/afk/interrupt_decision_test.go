package afk

import (
	"os"
	"syscall"
	"testing"
)

func TestInterruptShutdownStateDecisions(t *testing.T) {
	state := newInterruptShutdownState()

	if got := state.observe(syscall.SIGTERM); got != interruptDecisionIgnore {
		t.Fatalf("observe(SIGTERM) = %v, want ignore", got)
	}

	if got := state.observe(syscall.SIGINT); got != interruptDecisionStartGracefulShutdown {
		t.Fatalf("first observe(SIGINT) = %v, want graceful shutdown", got)
	}

	if got := state.observe(os.Interrupt); got != interruptDecisionForceHardShutdown {
		t.Fatalf("second observe(os.Interrupt) = %v, want hard shutdown", got)
	}
}

func TestInterruptShutdownStateKeepsIgnoringNonSigintDuringGracefulShutdown(t *testing.T) {
	state := newInterruptShutdownState()
	if got := state.observe(syscall.SIGINT); got != interruptDecisionStartGracefulShutdown {
		t.Fatalf("first observe(SIGINT) = %v, want graceful shutdown", got)
	}

	if got := state.observe(syscall.SIGTERM); got != interruptDecisionIgnore {
		t.Fatalf("observe(SIGTERM) during graceful shutdown = %v, want ignore", got)
	}

	if got := state.observe(syscall.SIGINT); got != interruptDecisionForceHardShutdown {
		t.Fatalf("second observe(SIGINT) = %v, want hard shutdown", got)
	}
}
