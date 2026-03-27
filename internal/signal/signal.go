package signal

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	mu     sync.Mutex
	hooks  []func()
	osExit = os.Exit
)

// SetExitFunc overrides the exit function called after force-kill hooks.
// Returns a restore function. Intended for testing only; this package is
// internal so the function is not visible to external consumers.
func SetExitFunc(fn func(int)) func() {
	mu.Lock()
	old := osExit
	osExit = fn
	mu.Unlock()
	return func() {
		mu.Lock()
		osExit = old
		mu.Unlock()
	}
}

// OnForceKill registers fn to be called when a second signal is received.
// Returns a function that deregisters the hook.
func OnForceKill(fn func()) func() {
	mu.Lock()
	hooks = append(hooks, fn)
	registered := len(hooks) - 1
	mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			mu.Lock()
			if registered < len(hooks) {
				hooks[registered] = nil
			}
			mu.Unlock()
		})
	}
}

func forceKill() {
	mu.Lock()
	snapshot := make([]func(), len(hooks))
	copy(snapshot, hooks)
	exitFn := osExit
	mu.Unlock()

	for _, fn := range snapshot {
		if fn != nil {
			fn()
		}
	}
	exitFn(1)
}

// NotifyContext returns a context that is cancelled when SIGINT or SIGTERM
// is received. On a second signal, registered force-kill hooks are called
// and the process exits with code 1. The returned cancel func stops
// listening for signals and should be deferred by the caller.
func NotifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	stopped := make(chan struct{})

	go func() {
		select {
		case <-ch:
			cancel()
		case <-stopped:
			return
		}

		// Context cancelled by first signal. Wait for second signal.
		select {
		case <-ch:
			forceKill()
		case <-stopped:
		}
	}()

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			signal.Stop(ch)
			close(stopped)
		})
		cancel()
	}
	return ctx, stop
}
