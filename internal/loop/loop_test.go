package loop

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

// fakeRunner records calls and returns preconfigured results.
type fakeRunner struct {
	mu      sync.Mutex
	calls   int
	results []runResult // per-call results; cycles if exhausted
	prompts []string
}

type runResult struct {
	exitCode int
	err      error
}

func (f *fakeRunner) Run(_ context.Context, prompt string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prompts = append(f.prompts, prompt)
	idx := f.calls
	f.calls++
	if idx < len(f.results) {
		return f.results[idx].exitCode, f.results[idx].err
	}
	last := f.results[len(f.results)-1]
	return last.exitCode, last.err
}

func (f *fakeRunner) getCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// spyLogger captures logged events and optionally returns errors.
type spyLogger struct {
	mu       sync.Mutex
	events   []loggedEvent
	failFrom int // if > 0, return error starting at this call index
}

type loggedEvent struct {
	event  string
	fields map[string]any
}

func (s *spyLogger) Log(event string, fields map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, loggedEvent{event: event, fields: fields})
	if s.failFrom > 0 && len(s.events) >= s.failFrom {
		return errors.New("log write failed")
	}
	return nil
}

func (s *spyLogger) eventNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, len(s.events))
	for i, e := range s.events {
		names[i] = e.event
	}
	return names
}

func TestSingleIteration(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0, err: nil}}}
	logger := &spyLogger{}
	cfg := Config{
		MaxIter: 1,
		Prompt:  "test prompt",
	}

	exitCode, err := Run(context.Background(), cfg, runner, logger, nil)
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
	cfg := Config{MaxIter: 3, Prompt: "p"}

	exitCode, err := Run(context.Background(), cfg, runner, logger, nil)
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
	cfg := Config{MaxIter: 2, Prompt: "p"}

	exitCode, err := Run(context.Background(), cfg, runner, logger, nil)
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
	cfg := Config{MaxIter: 2, Prompt: "p"}

	_, err := Run(context.Background(), cfg, runner, logger, nil)
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
	cfg := Config{MaxIter: 2, Prompt: "p"}

	exitCode, err := Run(context.Background(), cfg, runner, logger, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestAllNonZeroExitCodesExitOne(t *testing.T) {
	runner := &fakeRunner{results: []runResult{
		{exitCode: 1, err: nil},
	}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 3, Prompt: "p"}

	exitCode, err := Run(context.Background(), cfg, runner, logger, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 when all iterations have non-zero exit, got %d", exitCode)
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
	cfg := Config{MaxIter: 10, Prompt: "p"}

	exitCode, err := Run(ctx, cfg, cancellingRunner, logger, nil)
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

func TestIterationEventsBracketEachIteration(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 3, Prompt: "p"}

	_, err := Run(context.Background(), cfg, runner, logger, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := logger.eventNames()
	// Expect: session-start, (iteration-start, iteration-end) x3, session-end
	expected := []string{
		"session-start",
		"iteration-start", "iteration-end",
		"iteration-start", "iteration-end",
		"iteration-start", "iteration-end",
		"session-end",
	}
	if !slices.Equal(names, expected) {
		t.Fatalf("event sequence mismatch:\n  got:  %v\n  want: %v", names, expected)
	}
}

func TestIterationEndContainsExitCodeAndDuration(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 42}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 1, Prompt: "p"}

	_, err := Run(context.Background(), cfg, runner, logger, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the iteration-end event
	var iterEnd *loggedEvent
	for i := range logger.events {
		if logger.events[i].event == "iteration-end" {
			iterEnd = &logger.events[i]
			break
		}
	}
	if iterEnd == nil {
		t.Fatal("no iteration-end event found")
	}

	if iterEnd.fields["exitCode"] != 42 {
		t.Errorf("expected exitCode=42, got %v", iterEnd.fields["exitCode"])
	}
	if _, ok := iterEnd.fields["duration"]; !ok {
		t.Error("iteration-end missing duration field")
	}
	if iterEnd.fields["iteration"] != 1 {
		t.Errorf("expected iteration=1, got %v", iterEnd.fields["iteration"])
	}
}

func TestDaemonIterationNumberIncrements(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	runner := &cancellingFakeRunner{
		inner:       &fakeRunner{results: []runResult{{exitCode: 0}}},
		cancelAfter: 3,
		cancel:      cancel,
		calls:       &callCount,
	}
	logger := &spyLogger{}
	cfg := Config{
		Daemon:  true,
		Sleep:   10 * time.Millisecond,
		Prompt:  "p",
		MaxIter: 20,
	}

	_, err := Run(ctx, cfg, runner, logger, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Collect iteration numbers from iteration-start events
	var iterations []int
	logger.mu.Lock()
	for _, e := range logger.events {
		if e.event == "iteration-start" {
			iterations = append(iterations, e.fields["iteration"].(int))
		}
	}
	logger.mu.Unlock()

	if len(iterations) < 3 {
		t.Fatalf("expected at least 3 iterations, got %d", len(iterations))
	}
	for i, n := range iterations {
		if n != i+1 {
			t.Errorf("iteration %d: expected number %d, got %d", i, i+1, n)
		}
	}
}

func TestDaemonWakingEventLogged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	runner := &cancellingFakeRunner{
		inner:       &fakeRunner{results: []runResult{{exitCode: 0}}},
		cancelAfter: 2,
		cancel:      cancel,
		calls:       &callCount,
	}
	logger := &spyLogger{}
	cfg := Config{
		Daemon:  true,
		Sleep:   10 * time.Millisecond,
		Prompt:  "p",
		MaxIter: 20,
	}

	_, err := Run(ctx, cfg, runner, logger, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After first iteration and sleep, "waking" should be logged
	names := logger.eventNames()
	if !slices.Contains(names, "waking") {
		t.Errorf("expected waking event, got %v", names)
	}
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
	cfg := Config{
		Daemon:  true,
		Sleep:   10 * time.Millisecond,
		Prompt:  "p",
		MaxIter: 20,
	}

	exitCode, err := Run(ctx, cfg, runner, logger, nil)
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
	cfg := Config{
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
	exitCode, err := Run(ctx, cfg, runner, logger, nil)
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

// fakeWorkSource returns preconfigured work items.
type fakeWorkSource struct {
	items []workItem
	calls int
}

type workItem struct {
	prompt  string
	id      string
	title   string
	hasWork bool
	err     error
}

func (f *fakeWorkSource) Next() (prompt string, issueID string, issueTitle string, ok bool, err error) {
	idx := f.calls
	f.calls++
	if idx < len(f.items) {
		w := f.items[idx]
		return w.prompt, w.id, w.title, w.hasWork, w.err
	}
	return "", "", "", false, nil
}

func TestWorkSource_UsesReturnedPrompt(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 1, Prompt: "static prompt"}
	ws := &fakeWorkSource{items: []workItem{
		{prompt: "dynamic prompt", id: "afk-1", title: "Fix bug", hasWork: true},
	}}

	exitCode, err := Run(context.Background(), cfg, runner, logger, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if runner.prompts[0] != "dynamic prompt" {
		t.Errorf("expected dynamic prompt, got %q", runner.prompts[0])
	}
}

func TestWorkSource_NoWorkExitsMaxIter(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 5, Prompt: "p"}
	ws := &fakeWorkSource{items: []workItem{
		{hasWork: false},
	}}

	exitCode, err := Run(context.Background(), cfg, runner, logger, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if runner.calls != 0 {
		t.Errorf("expected 0 runner calls, got %d", runner.calls)
	}
	// Should have session-end with reason=no-work
	lastEvent := logger.events[len(logger.events)-1]
	if lastEvent.fields["reason"] != "no-work" {
		t.Errorf("expected reason=no-work, got %v", lastEvent.fields["reason"])
	}
}

func TestWorkSource_DaemonRetriesOnNoWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 20, Daemon: true, Sleep: 10 * time.Millisecond, Prompt: "p"}
	ws := &fakeWorkSource{items: []workItem{
		{hasWork: false},
		{prompt: "work", id: "afk-1", title: "Fix", hasWork: true},
	}}

	// Cancel after the runner gets called once
	go func() {
		for {
			time.Sleep(5 * time.Millisecond)
			if runner.getCalls() >= 1 {
				cancel()
				return
			}
		}
	}()

	exitCode, err := Run(ctx, cfg, runner, logger, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	// Runner should have been called with the work from the second Next() call
	if runner.getCalls() < 1 {
		t.Errorf("expected at least 1 runner call, got %d", runner.calls)
	}
	if runner.prompts[0] != "work" {
		t.Errorf("expected prompt %q, got %q", "work", runner.prompts[0])
	}
}

func TestWorkSource_BeadsCheckLogged(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 1, Prompt: "p"}
	ws := &fakeWorkSource{items: []workItem{
		{prompt: "work", id: "afk-1", title: "Fix", hasWork: true},
	}}

	_, err := Run(context.Background(), cfg, runner, logger, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, e := range logger.events {
		if e.event == "beads-check" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected beads-check event, got events: %v", logger.eventNames())
	}
}

func TestWorkSource_IterationLogsIncludeIssueInfo(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 1, Prompt: "p"}
	ws := &fakeWorkSource{items: []workItem{
		{prompt: "work", id: "afk-1", title: "Fix bug", hasWork: true},
	}}

	_, err := Run(context.Background(), cfg, runner, logger, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, e := range logger.events {
		if e.event == "iteration-start" {
			if e.fields["issueID"] != "afk-1" {
				t.Errorf("iteration-start issueID = %v, want afk-1", e.fields["issueID"])
			}
			if e.fields["issueTitle"] != "Fix bug" {
				t.Errorf("iteration-start issueTitle = %v, want Fix bug", e.fields["issueTitle"])
			}
		}
		if e.event == "iteration-end" {
			if e.fields["issueID"] != "afk-1" {
				t.Errorf("iteration-end issueID = %v, want afk-1", e.fields["issueID"])
			}
			if e.fields["issueTitle"] != "Fix bug" {
				t.Errorf("iteration-end issueTitle = %v, want Fix bug", e.fields["issueTitle"])
			}
		}
	}
}

func TestWorkSource_ErrorExitsNonZeroMaxIter(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 5, Prompt: "p"}
	ws := &fakeWorkSource{items: []workItem{
		{err: errors.New("bd: connection refused")},
	}}

	exitCode, err := Run(context.Background(), cfg, runner, logger, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if runner.calls != 0 {
		t.Errorf("expected 0 runner calls, got %d", runner.calls)
	}
	// Should log work-source-error
	found := false
	for _, e := range logger.events {
		if e.event == "work-source-error" {
			found = true
			if e.fields["err"] != "bd: connection refused" {
				t.Errorf("expected error message in log, got %v", e.fields["err"])
			}
			break
		}
	}
	if !found {
		t.Errorf("expected work-source-error event, got events: %v", logger.eventNames())
	}
	// Session-end reason should be work-source-error
	lastEvent := logger.events[len(logger.events)-1]
	if lastEvent.fields["reason"] != "work-source-error" {
		t.Errorf("expected reason=work-source-error, got %v", lastEvent.fields["reason"])
	}
}

func TestWorkSource_ErrorDaemonRetriesAfterSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 20, Daemon: true, Sleep: 10 * time.Millisecond, Prompt: "p"}
	ws := &fakeWorkSource{items: []workItem{
		{err: errors.New("bd: timeout")},
		{prompt: "work", id: "afk-1", title: "Fix", hasWork: true},
	}}

	// Cancel after the runner gets called once
	go func() {
		for {
			time.Sleep(5 * time.Millisecond)
			if runner.getCalls() >= 1 {
				cancel()
				return
			}
		}
	}()

	exitCode, err := Run(ctx, cfg, runner, logger, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	// Should have logged the error then retried successfully
	foundErr := false
	for _, e := range logger.events {
		if e.event == "work-source-error" {
			foundErr = true
			break
		}
	}
	if !foundErr {
		t.Errorf("expected work-source-error event, got events: %v", logger.eventNames())
	}
	if runner.getCalls() < 1 {
		t.Errorf("expected at least 1 runner call after retry, got %d", runner.getCalls())
	}
}

func TestWorkSource_NilPreservesExistingBehaviour(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	logger := &spyLogger{}
	cfg := Config{MaxIter: 2, Prompt: "static"}

	exitCode, err := Run(context.Background(), cfg, runner, logger, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if runner.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", runner.calls)
	}
	if runner.prompts[0] != "static" {
		t.Errorf("expected static prompt, got %q", runner.prompts[0])
	}

	// No beads-check events should be logged
	for _, e := range logger.events {
		if e.event == "beads-check" {
			t.Error("beads-check should not be logged when WorkSource is nil")
		}
	}
}

func TestLogFailureExitsMaxIter(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	// Fail on second Log call (after session-start)
	logger := &spyLogger{failFrom: 2}
	cfg := Config{MaxIter: 5, Prompt: "p"}

	exitCode, err := Run(context.Background(), cfg, runner, logger, nil)
	if err == nil {
		t.Fatal("expected error from log failure, got nil")
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	// Should not have run all 5 iterations
	if runner.getCalls() > 1 {
		t.Errorf("expected loop to stop early, but runner got %d calls", runner.getCalls())
	}
}

func TestDaemonAllFailuresExitOne(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &fakeRunner{results: []runResult{
		{exitCode: 1, err: errors.New("fail")},
	}}
	callCount := 0
	cancellingRunner := &cancellingFakeRunner{
		inner:       runner,
		cancelAfter: 3,
		cancel:      cancel,
		calls:       &callCount,
	}
	logger := &spyLogger{}
	cfg := Config{
		Daemon:  true,
		Sleep:   10 * time.Millisecond,
		Prompt:  "p",
		MaxIter: 20,
	}

	exitCode, err := Run(ctx, cfg, cancellingRunner, logger, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 when all daemon iterations fail, got %d", exitCode)
	}
}

func TestDaemonPartialSuccessExitZero(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &fakeRunner{results: []runResult{
		{exitCode: 1, err: errors.New("fail")},
		{exitCode: 0, err: nil},
	}}
	callCount := 0
	cancellingRunner := &cancellingFakeRunner{
		inner:       runner,
		cancelAfter: 2,
		cancel:      cancel,
		calls:       &callCount,
	}
	logger := &spyLogger{}
	cfg := Config{
		Daemon:  true,
		Sleep:   10 * time.Millisecond,
		Prompt:  "p",
		MaxIter: 20,
	}

	exitCode, err := Run(ctx, cfg, cancellingRunner, logger, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 when some daemon iterations succeed, got %d", exitCode)
	}
}

func TestLogFailureExitsDaemon(t *testing.T) {
	runner := &fakeRunner{results: []runResult{{exitCode: 0}}}
	// Fail on second Log call (after session-start)
	logger := &spyLogger{failFrom: 2}
	cfg := Config{MaxIter: 20, Daemon: true, Sleep: 10 * time.Millisecond, Prompt: "p"}

	exitCode, err := Run(context.Background(), cfg, runner, logger, nil)
	if err == nil {
		t.Fatal("expected error from log failure, got nil")
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}
