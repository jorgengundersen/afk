package loop

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jorgengundersen/afk/internal/config"
)

// fakeRunner records calls and returns preconfigured results.
type fakeRunner struct {
	calls   int
	results []runResult // per-call results; cycles if exhausted
	prompts []string
}

type runResult struct {
	exitCode int
	err      error
}

func (f *fakeRunner) Run(_ context.Context, prompt string) (int, error) {
	f.prompts = append(f.prompts, prompt)
	idx := f.calls
	f.calls++
	if idx < len(f.results) {
		return f.results[idx].exitCode, f.results[idx].err
	}
	last := f.results[len(f.results)-1]
	return last.exitCode, last.err
}

// spyLogger captures logged events.
type spyLogger struct {
	events []loggedEvent
}

type loggedEvent struct {
	event  string
	fields map[string]any
}

func (s *spyLogger) Log(event string, fields map[string]any) {
	s.events = append(s.events, loggedEvent{event: event, fields: fields})
}

func (s *spyLogger) eventNames() []string {
	names := make([]string, len(s.events))
	for i, e := range s.events {
		names[i] = e.event
	}
	return names
}

func TestSingleIteration(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0, err: nil}}}
	logger := &spyLogger{}
	cfg := config.Config{
		MaxIter: 1,
		Prompt:  "test prompt",
	}

	exitCode, err := Run(context.Background(), cfg, runner, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if runner.calls != 1 {
		t.Fatalf("expected 1 call, got %d", runner.calls)
	}
	if runner.prompts[0] != "test prompt" {
		t.Fatalf("expected prompt %q, got %q", "test prompt", runner.prompts[0])
	}

	// Verify log events
	names := logger.eventNames()
	if len(names) < 3 {
		t.Fatalf("expected at least 3 events, got %d: %v", len(names), names)
	}
	if names[0] != "session-start" {
		t.Errorf("first event should be session-start, got %q", names[0])
	}
	if names[len(names)-1] != "session-end" {
		t.Errorf("last event should be session-end, got %q", names[len(names)-1])
	}
	// Check session-end has reason=complete
	lastEvent := logger.events[len(logger.events)-1]
	if lastEvent.fields["reason"] != "complete" {
		t.Errorf("session-end reason should be 'complete', got %v", lastEvent.fields["reason"])
	}
}

func TestMultipleIterations(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := config.Config{MaxIter: 3, Prompt: "p"}

	exitCode, err := Run(context.Background(), cfg, runner, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if runner.calls != 3 {
		t.Fatalf("expected 3 calls, got %d", runner.calls)
	}
}

func TestNonZeroExitContinues(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 1}, {exitCode: 0}}}
	logger := &spyLogger{}
	cfg := config.Config{MaxIter: 2, Prompt: "p"}

	exitCode, err := Run(context.Background(), cfg, runner, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if runner.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", runner.calls)
	}
}

func TestLaunchFailureContinues(t *testing.T) {
	runner := &fakeRunner{results: []runResult{
		{exitCode: 0, err: errors.New("launch failed")},
		{exitCode: 0, err: errors.New("launch failed")},
	}}
	logger := &spyLogger{}
	cfg := config.Config{MaxIter: 2, Prompt: "p"}

	_, err := Run(context.Background(), cfg, runner, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", runner.calls)
	}
	// Verify error events were logged
	names := logger.eventNames()
	errorCount := 0
	for _, n := range names {
		if n == "error" {
			errorCount++
		}
	}
	if errorCount != 2 {
		t.Errorf("expected 2 error events, got %d", errorCount)
	}
}

func TestAllLaunchFailuresExitOne(t *testing.T) {
	runner := &fakeRunner{results: []runResult{
		{exitCode: 0, err: errors.New("fail")},
	}}
	logger := &spyLogger{}
	cfg := config.Config{MaxIter: 2, Prompt: "p"}

	exitCode, err := Run(context.Background(), cfg, runner, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestContextCancellationMidLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	// Override Run to cancel after 2 iterations
	cancellingRunner := &cancellingFakeRunner{
		inner:       runner,
		cancelAfter: 2,
		cancel:      cancel,
		calls:       &callCount,
	}
	logger := &spyLogger{}
	cfg := config.Config{MaxIter: 10, Prompt: "p"}

	exitCode, err := Run(ctx, cfg, cancellingRunner, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if callCount > 3 {
		t.Fatalf("expected ~2 calls, got %d", callCount)
	}
	// Last event should be session-end with reason=signal
	names := logger.eventNames()
	lastEvent := logger.events[len(names)-1]
	if lastEvent.fields["reason"] != "signal" {
		t.Errorf("expected reason=signal, got %v", lastEvent.fields["reason"])
	}
}

type cancellingFakeRunner struct {
	inner       *fakeRunner
	cancelAfter int
	cancel      context.CancelFunc
	calls       *int
}

func (c *cancellingFakeRunner) Run(ctx context.Context, prompt string) (int, error) {
	*c.calls++
	code, err := c.inner.Run(ctx, prompt)
	if *c.calls >= c.cancelAfter {
		c.cancel()
	}
	return code, err
}

func TestDaemonSleepsBetweenIterations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	runner := &cancellingFakeRunner{
		inner:       &fakeRunner{results: []runResult{{exitCode: 0}}},
		cancelAfter: 2,
		cancel:      cancel,
		calls:       &callCount,
	}
	logger := &spyLogger{}
	cfg := config.Config{
		Daemon:  true,
		Sleep:   10 * time.Millisecond,
		Prompt:  "p",
		MaxIter: 20,
	}

	exitCode, err := Run(ctx, cfg, runner, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount)
	}
	// Verify sleeping event was logged
	names := logger.eventNames()
	if !slices.Contains(names, "sleeping") {
		t.Errorf("expected sleeping event, got %v", names)
	}
}

func TestDaemonSleepInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := config.Config{
		Daemon:  true,
		Sleep:   10 * time.Second, // long sleep
		Prompt:  "p",
		MaxIter: 20,
	}

	// Cancel during sleep (after first iteration)
	go func() {
		// Wait for at least one iteration to complete
		for {
			time.Sleep(5 * time.Millisecond)
			names := logger.eventNames()
			if slices.Contains(names, "sleeping") {
				cancel()
				return
			}
		}
	}()

	start := time.Now()
	exitCode, err := Run(ctx, cfg, runner, logger)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	// Should exit well before the 10s sleep
	if elapsed > 2*time.Second {
		t.Fatalf("sleep was not interrupted promptly, took %v", elapsed)
	}
	// Last event should be session-end with reason=signal
	lastEvent := logger.events[len(logger.events)-1]
	if lastEvent.fields["reason"] != "signal" {
		t.Errorf("expected reason=signal, got %v", lastEvent.fields["reason"])
	}
}
