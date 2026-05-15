package afk

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestInterruptibleSleepWakesOnSigint(t *testing.T) {
	interrupts := make(chan os.Signal, 1)

	go func() {
		time.Sleep(25 * time.Millisecond)
		interrupts <- syscall.SIGINT
	}()

	startedAt := time.Now()
	interrupted := interruptibleSleep(500*time.Millisecond, interrupts)
	elapsed := time.Since(startedAt)

	if !interrupted {
		t.Fatalf("interruptibleSleep() interrupted = false, want true")
	}
	if elapsed >= 200*time.Millisecond {
		t.Fatalf("interruptibleSleep() elapsed = %v, want < 200ms after SIGINT", elapsed)
	}
}
