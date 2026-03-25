package signal

import (
	"context"
	"os/signal"
	"syscall"
)

// NotifyContext returns a context that is cancelled when SIGINT or SIGTERM
// is received. The returned cancel func stops listening for signals and
// should be deferred by the caller.
func NotifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
}
