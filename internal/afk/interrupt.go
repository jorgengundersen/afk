package afk

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

func interruptibleSleep(duration time.Duration, interrupts <-chan os.Signal) bool {
	if duration <= 0 {
		return false
	}

	if interrupts == nil {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT)
		defer signal.Stop(sigCh)
		interrupts = sigCh
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return false
	case sig := <-interrupts:
		return sig == syscall.SIGINT || sig == os.Interrupt
	}
}
