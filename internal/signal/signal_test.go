package signal_test

import (
	"context"
	"os"
	ossignal "os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/jorgengundersen/afk/internal/signal"
)

func TestCancelFuncStopsListening(t *testing.T) {
	ctx, cancel := signal.NotifyContext(context.Background())

	// Stop listening before sending the signal.
	cancel()

	// Catch SIGINT ourselves so the default handler doesn't kill the process.
	ch := make(chan os.Signal, 1)
	ossignal.Notify(ch, syscall.SIGINT)
	defer ossignal.Stop(ch)

	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	// Drain the signal so the test doesn't leak.
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("SIGINT not received by fallback handler")
	}

	// Context was cancelled by cancel(), not by the signal.
	if ctx.Err() != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", ctx.Err())
	}
}

func TestContextCancelledOnSIGINT(t *testing.T) {
	ctx, cancel := signal.NotifyContext(context.Background())
	defer cancel()

	// Send SIGINT to ourselves after a short delay.
	go func() {
		time.Sleep(10 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	select {
	case <-ctx.Done():
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after SIGINT")
	}
}

func TestContextCancelledOnSIGTERM(t *testing.T) {
	ctx, cancel := signal.NotifyContext(context.Background())
	defer cancel()

	go func() {
		time.Sleep(10 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	select {
	case <-ctx.Done():
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after SIGTERM")
	}
}

func TestSecondSignalCallsForceKill(t *testing.T) {
	// Override os.Exit so the test process survives.
	restore := signal.SetExitFunc(func(int) {})
	defer restore()

	ctx, cancel := signal.NotifyContext(context.Background())
	defer cancel()

	called := make(chan struct{}, 1)
	deregister := signal.OnForceKill(func() {
		called <- struct{}{}
	})
	defer deregister()

	// First signal cancels context.
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after first SIGINT")
	}

	// Second signal should call the force-kill hook.
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	select {
	case <-called:
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("force-kill hook not called after second signal")
	}
}

func TestForceKillDeregister(t *testing.T) {
	restore := signal.SetExitFunc(func(int) {})
	defer restore()

	ctx, cancel := signal.NotifyContext(context.Background())
	defer cancel()

	called := false
	deregister := signal.OnForceKill(func() {
		called = true
	})

	// Deregister before second signal.
	deregister()

	// Catch signals so the test process doesn't die.
	ch := make(chan os.Signal, 1)
	ossignal.Notify(ch, syscall.SIGINT)
	defer ossignal.Stop(ch)

	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	<-ctx.Done()

	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	// Drain signal.
	select {
	case <-ch:
	case <-time.After(time.Second):
	}

	time.Sleep(50 * time.Millisecond)
	if called {
		t.Fatal("deregistered hook was called")
	}
}
