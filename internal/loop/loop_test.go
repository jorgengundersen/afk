package loop

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jorgengundersen/afk/internal/beads"
	"github.com/jorgengundersen/afk/internal/config"
	"github.com/jorgengundersen/afk/internal/prompt"
)

// fakeHarness records calls and returns preconfigured results.
type fakeHarness struct {
	prompt   string
	exitCode int
	err      error
}

func (f *fakeHarness) Run(_ context.Context, prompt string) (int, error) {
	f.prompt = prompt
	return f.exitCode, f.err
}

// fakeBeads records calls and returns preconfigured results.
type fakeBeads struct {
	called bool
	issues []beads.Issue
	err    error
}

func (f *fakeBeads) Ready(_ context.Context) ([]beads.Issue, error) {
	f.called = true
	return f.issues, f.err
}

// fakeLogger records logged events.
type fakeLogger struct {
	events []loggedEvent
}

type loggedEvent struct {
	name   string
	fields []Field
}

func (f *fakeLogger) Event(name string, fields ...Field) {
	f.events = append(f.events, loggedEvent{name: name, fields: fields})
}

func TestRunOnce_HappyPath(t *testing.T) {
	h := &fakeHarness{}
	b := &fakeBeads{issues: []beads.Issue{{ID: "TST-1", Title: "Fix bug"}}}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled: true,
		Prompt:       "do the work",
	}

	ran, err := RunOnce(context.Background(), cfg, h, b, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Fatal("expected ranWork=true")
	}
	if !b.called {
		t.Fatal("expected beads.Ready to be called")
	}
	if h.prompt == "" {
		t.Fatal("expected harness to receive a prompt")
	}

	// Should have iteration_start and iteration_end events
	if len(l.events) < 2 {
		t.Fatalf("expected at least 2 log events, got %d", len(l.events))
	}
	if l.events[0].name != "iteration-start" {
		t.Errorf("first event = %q, want iteration-start", l.events[0].name)
	}
	last := l.events[len(l.events)-1]
	if last.name != "iteration-end" {
		t.Errorf("last event = %q, want iteration-end", last.name)
	}
	if !hasField(last.fields, "status", "ok") {
		t.Errorf("iteration_end missing status=ok, fields: %v", last.fields)
	}
}

func TestRunOnce_NoWork(t *testing.T) {
	h := &fakeHarness{}
	b := &fakeBeads{err: beads.ErrNoWork}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled: true,
		Prompt:       "",
	}

	ran, err := RunOnce(context.Background(), cfg, h, b, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ran {
		t.Fatal("expected ranWork=false when no work")
	}
	if h.prompt != "" {
		t.Fatal("harness should not have been called")
	}

	last := l.events[len(l.events)-1]
	if !hasField(last.fields, "status", "no_work") {
		t.Errorf("expected status=no_work, got fields: %v", last.fields)
	}
}

func TestRunOnce_HarnessFailure(t *testing.T) {
	h := &fakeHarness{exitCode: 1, err: errors.New("exit 1")}
	b := &fakeBeads{issues: []beads.Issue{{ID: "TST-1", Title: "Fix bug"}}}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled: true,
		Prompt:       "do it",
	}

	ran, err := RunOnce(context.Background(), cfg, h, b, l)
	if err == nil {
		t.Fatal("expected error from harness failure")
	}
	if !ran {
		t.Fatal("expected ranWork=true even on harness failure (work was attempted)")
	}

	last := l.events[len(l.events)-1]
	if !hasField(last.fields, "status", "fail") {
		t.Errorf("expected status=fail, got fields: %v", last.fields)
	}
}

func TestRunOnce_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	h := &fakeHarness{}
	b := &fakeBeads{issues: []beads.Issue{{ID: "TST-1", Title: "Fix bug"}}}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled: true,
		Prompt:       "do it",
	}

	_, err := RunOnce(ctx, cfg, h, b, l)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestRunOnce_BeadsDisabled(t *testing.T) {
	h := &fakeHarness{}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled: false,
		Prompt:       "just do the prompt",
	}

	ran, err := RunOnce(context.Background(), cfg, h, nil, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Fatal("expected ranWork=true")
	}
	if h.prompt != "just do the prompt" {
		t.Errorf("prompt = %q, want %q", h.prompt, "just do the prompt")
	}

	last := l.events[len(l.events)-1]
	if !hasField(last.fields, "status", "ok") {
		t.Errorf("expected status=ok, got fields: %v", last.fields)
	}
}

// multiHarness returns different results per call.
type multiHarness struct {
	calls     int
	exitCodes []int
	errs      []error
}

func (m *multiHarness) Run(_ context.Context, _ string) (int, error) {
	i := m.calls
	m.calls++
	if i < len(m.exitCodes) {
		return m.exitCodes[i], m.errs[i]
	}
	return 0, nil
}

// multiBeads returns different results per call.
type multiBeads struct {
	calls   int
	results []beadsResult
}

type beadsResult struct {
	issues []beads.Issue
	err    error
}

func (m *multiBeads) Ready(_ context.Context) ([]beads.Issue, error) {
	i := m.calls
	m.calls++
	if i < len(m.results) {
		return m.results[i].issues, m.results[i].err
	}
	return nil, beads.ErrNoWork
}

func TestRunMaxIter_SingleSuccess(t *testing.T) {
	h := &multiHarness{exitCodes: []int{0}, errs: []error{nil}}
	b := &multiBeads{results: []beadsResult{
		{issues: []beads.Issue{{ID: "T-1", Title: "Fix"}}, err: nil},
	}}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.MaxIterationsMode,
		MaxIterations: 1,
		BeadsEnabled:  true,
		Prompt:        "do it",
	}

	err := RunMaxIter(context.Background(), cfg, h, b, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.calls != 1 {
		t.Fatalf("expected 1 harness call, got %d", h.calls)
	}

	// Check session-start and session-end events
	if l.events[0].name != "session-start" {
		t.Errorf("first event = %q, want session-start", l.events[0].name)
	}
	last := l.events[len(l.events)-1]
	if last.name != "session-end" {
		t.Errorf("last event = %q, want session-end", last.name)
	}
	if !hasField(last.fields, "total", "1") {
		t.Errorf("expected total=1, got %v", last.fields)
	}
	if !hasField(last.fields, "succeeded", "1") {
		t.Errorf("expected succeeded=1, got %v", last.fields)
	}
	if !hasField(last.fields, "failed", "0") {
		t.Errorf("expected failed=0, got %v", last.fields)
	}
}

func TestRunMaxIter_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first harness call.
	callCount := 0
	cancelHarness := &cancellingHarness{cancel: cancel, callCount: &callCount}

	issue := beads.Issue{ID: "T-1", Title: "Fix"}
	b := &multiBeads{results: []beadsResult{
		{issues: []beads.Issue{issue}},
		{issues: []beads.Issue{issue}},
		{issues: []beads.Issue{issue}},
	}}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.MaxIterationsMode,
		MaxIterations: 10,
		BeadsEnabled:  true,
		Prompt:        "do it",
	}

	err := RunMaxIter(ctx, cfg, cancelHarness, b, l)
	if err != nil {
		t.Fatalf("expected nil on context cancel, got: %v", err)
	}
	if *cancelHarness.callCount > 2 {
		t.Fatalf("expected at most 2 harness calls, got %d", *cancelHarness.callCount)
	}

	last := l.events[len(l.events)-1]
	if last.name != "session-end" {
		t.Errorf("last event = %q, want session-end", last.name)
	}
}

// cancellingHarness cancels the context after the first call.
type cancellingHarness struct {
	cancel    context.CancelFunc
	callCount *int
}

func (c *cancellingHarness) Run(_ context.Context, _ string) (int, error) {
	*c.callCount++
	if *c.callCount == 1 {
		c.cancel()
	}
	return 0, nil
}

func TestRunMaxIter_AllFailed(t *testing.T) {
	h := &multiHarness{
		exitCodes: []int{1, 1},
		errs:      []error{errors.New("exit 1"), errors.New("exit 1")},
	}
	issue := beads.Issue{ID: "T-1", Title: "Fix"}
	b := &multiBeads{results: []beadsResult{
		{issues: []beads.Issue{issue}},
		{issues: []beads.Issue{issue}},
	}}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.MaxIterationsMode,
		MaxIterations: 2,
		BeadsEnabled:  true,
		Prompt:        "do it",
	}

	err := RunMaxIter(context.Background(), cfg, h, b, l)
	if !errors.Is(err, ErrAllFailed) {
		t.Fatalf("expected ErrAllFailed, got: %v", err)
	}

	last := l.events[len(l.events)-1]
	if !hasField(last.fields, "failed", "2") {
		t.Errorf("expected failed=2, got %v", last.fields)
	}
	if !hasField(last.fields, "succeeded", "0") {
		t.Errorf("expected succeeded=0, got %v", last.fields)
	}
}

func TestRunMaxIter_EarlyStopNoWork(t *testing.T) {
	h := &multiHarness{
		exitCodes: []int{0},
		errs:      []error{nil},
	}
	b := &multiBeads{results: []beadsResult{
		{issues: []beads.Issue{{ID: "T-1", Title: "Fix"}}},
		{err: beads.ErrNoWork}, // second call: no work
	}}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.MaxIterationsMode,
		MaxIterations: 5,
		BeadsEnabled:  true,
		Prompt:        "",
	}

	err := RunMaxIter(context.Background(), cfg, h, b, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.calls != 1 {
		t.Fatalf("expected 1 harness call (stop after no work), got %d", h.calls)
	}

	last := l.events[len(l.events)-1]
	if !hasField(last.fields, "total", "1") {
		t.Errorf("expected total=1, got %v", last.fields)
	}
}

func TestRunMaxIter_NIterations(t *testing.T) {
	h := &multiHarness{
		exitCodes: []int{0, 0, 0},
		errs:      []error{nil, nil, nil},
	}
	issue := beads.Issue{ID: "T-1", Title: "Fix"}
	b := &multiBeads{results: []beadsResult{
		{issues: []beads.Issue{issue}},
		{issues: []beads.Issue{issue}},
		{issues: []beads.Issue{issue}},
	}}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.MaxIterationsMode,
		MaxIterations: 3,
		BeadsEnabled:  true,
		Prompt:        "do it",
	}

	err := RunMaxIter(context.Background(), cfg, h, b, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.calls != 3 {
		t.Fatalf("expected 3 harness calls, got %d", h.calls)
	}

	last := l.events[len(l.events)-1]
	if !hasField(last.fields, "total", "3") {
		t.Errorf("expected total=3, got %v", last.fields)
	}
	if !hasField(last.fields, "succeeded", "3") {
		t.Errorf("expected succeeded=3, got %v", last.fields)
	}
}

func TestRunDaemon_CleanShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately so daemon exits on first context check.
	cancel()

	l := &fakeLogger{}
	h := &fakeHarness{}
	cfg := config.Config{
		Mode:          config.DaemonMode,
		SleepInterval: time.Millisecond,
		Prompt:        "do it",
		Harness:       "claude",
	}

	err := RunDaemon(ctx, cfg, h, nil, l)
	if err != nil {
		t.Fatalf("expected nil on clean shutdown, got: %v", err)
	}

	// Must have session-start and session-end events.
	if len(l.events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(l.events))
	}
	if l.events[0].name != "session-start" {
		t.Errorf("first event = %q, want session-start", l.events[0].name)
	}
	last := l.events[len(l.events)-1]
	if last.name != "session-end" {
		t.Errorf("last event = %q, want session-end", last.name)
	}
}

func TestRunDaemon_SleepWakeCycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Beads returns no work twice, then cancel.
	b := &multiBeads{results: []beadsResult{
		{err: beads.ErrNoWork},
		{err: beads.ErrNoWork},
	}}
	h := &fakeHarness{}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.DaemonMode,
		SleepInterval: time.Millisecond,
		BeadsEnabled:  true,
		Prompt:        "",
		Harness:       "claude",
	}

	// Cancel after 2 sleep/wake cycles by watching log events.
	wakeCount := 0
	countingLogger := &callbackLogger{
		inner: l,
		onEvent: func(name string) {
			if name == "waking" {
				wakeCount++
				if wakeCount >= 2 {
					cancel()
				}
			}
		},
	}

	err := RunDaemon(ctx, cfg, h, b, countingLogger)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}

	// Verify sleeping and waking events were logged.
	var sleepCount, wakeEvtCount int
	for _, e := range countingLogger.inner.events {
		switch e.name {
		case "sleeping":
			sleepCount++
			if !hasField(e.fields, "duration", "1ms") {
				t.Errorf("sleeping event missing duration=1ms, got %v", e.fields)
			}
		case "waking":
			wakeEvtCount++
		}
	}
	if sleepCount < 2 {
		t.Errorf("expected at least 2 sleeping events, got %d", sleepCount)
	}
	if wakeEvtCount < 2 {
		t.Errorf("expected at least 2 waking events, got %d", wakeEvtCount)
	}
}

func TestRunDaemon_ImmediateRecheckAfterWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// First call: work available. Second call: work available. Third: cancel.
	issue := beads.Issue{ID: "T-1", Title: "Fix"}
	b := &multiBeads{results: []beadsResult{
		{issues: []beads.Issue{issue}},
		{issues: []beads.Issue{issue}},
	}}
	harnessCallCount := 0
	cancelAfterTwo := &callbackHarness{
		onRun: func() {
			harnessCallCount++
			if harnessCallCount >= 2 {
				cancel()
			}
		},
	}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.DaemonMode,
		SleepInterval: 10 * time.Second, // Long sleep — should NOT be hit.
		BeadsEnabled:  true,
		Prompt:        "do it",
		Harness:       "claude",
	}

	start := time.Now()
	err := RunDaemon(ctx, cfg, cancelAfterTwo, b, l)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if harnessCallCount < 2 {
		t.Fatalf("expected at least 2 harness calls, got %d", harnessCallCount)
	}
	// If sleep was triggered, this would take >=10s. Should be near-instant.
	if elapsed > 5*time.Second {
		t.Fatalf("took %v — sleep should not have been triggered between work iterations", elapsed)
	}

	// Verify no sleeping events were logged.
	for _, e := range l.events {
		if e.name == "sleeping" {
			t.Error("sleeping event should not be logged when work is available")
		}
	}
}

func TestRunDaemon_ContextCancelDuringSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Return no work so daemon enters sleep with a long interval.
	b := &fakeBeads{err: beads.ErrNoWork}
	h := &fakeHarness{}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.DaemonMode,
		SleepInterval: 10 * time.Minute, // Very long — must be interrupted by cancel.
		BeadsEnabled:  true,
		Prompt:        "",
		Harness:       "claude",
	}

	// Cancel shortly after daemon enters sleep.
	sleepLogger := &callbackLogger{
		inner: l,
		onEvent: func(name string) {
			if name == "sleeping" {
				go func() {
					time.Sleep(5 * time.Millisecond)
					cancel()
				}()
			}
		},
	}

	start := time.Now()
	err := RunDaemon(ctx, cfg, h, b, sleepLogger)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	// Should exit quickly, not wait for 10 minutes.
	if elapsed > 5*time.Second {
		t.Fatalf("took %v — context cancel during sleep should exit immediately", elapsed)
	}

	// daemon_stop must be the last event.
	last := sleepLogger.inner.events[len(sleepLogger.inner.events)-1]
	if last.name != "session-end" {
		t.Errorf("last event = %q, want session-end", last.name)
	}
}

// callbackHarness calls a callback on each Run invocation.
type callbackHarness struct {
	onRun func()
}

func (c *callbackHarness) Run(_ context.Context, _ string) (int, error) {
	if c.onRun != nil {
		c.onRun()
	}
	return 0, nil
}

// callbackLogger wraps fakeLogger and fires a callback on each event.
type callbackLogger struct {
	inner   *fakeLogger
	onEvent func(name string)
}

func (c *callbackLogger) Event(name string, fields ...Field) {
	c.inner.Event(name, fields...)
	if c.onEvent != nil {
		c.onEvent(name)
	}
}

func hasField(fields []Field, key, value string) bool {
	for _, f := range fields {
		if f.Key == key && f.Value == value {
			return true
		}
	}
	return false
}

func TestRunOnce_BeadsCheckEvent(t *testing.T) {
	h := &fakeHarness{}
	b := &fakeBeads{issues: []beads.Issue{{ID: "TST-1", Title: "Fix bug"}, {ID: "TST-2", Title: "Add feature"}}}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled: true,
		Prompt:       "do work",
	}

	_, err := RunOnce(context.Background(), cfg, h, b, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have a beads-check event with count after iteration-start
	var found bool
	for _, e := range l.events {
		if e.name == "beads-check" {
			found = true
			if !hasField(e.fields, "count", "2") {
				t.Errorf("beads-check event missing count=2, got %v", e.fields)
			}
		}
	}
	if !found {
		t.Error("expected beads-check event to be logged")
	}
}

func TestRunOnce_BeadsCheckEventNoWork(t *testing.T) {
	h := &fakeHarness{}
	b := &fakeBeads{err: beads.ErrNoWork}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled: true,
		Prompt:       "",
	}

	RunOnce(context.Background(), cfg, h, b, l)

	var found bool
	for _, e := range l.events {
		if e.name == "beads-check" {
			found = true
			if !hasField(e.fields, "count", "0") {
				t.Errorf("beads-check event missing count=0, got %v", e.fields)
			}
		}
	}
	if !found {
		t.Error("expected beads-check event even when no work")
	}
}

func TestRunDaemon_ErrorEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Harness fails on first call, then cancel.
	h := &multiHarness{
		exitCodes: []int{1},
		errs:      []error{errors.New("exit 1")},
	}
	b := &multiBeads{results: []beadsResult{
		{issues: []beads.Issue{{ID: "T-1", Title: "Fix"}}},
	}}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.DaemonMode,
		SleepInterval: time.Millisecond,
		BeadsEnabled:  true,
		Prompt:        "do it",
		Harness:       "claude",
	}

	// Cancel after first iteration-end so daemon exits.
	countingLogger := &callbackLogger{
		inner: l,
		onEvent: func(name string) {
			if name == "iteration-end" {
				cancel()
			}
		},
	}

	err := RunDaemon(ctx, cfg, h, b, countingLogger)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}

	var found bool
	for _, e := range l.events {
		if e.name == "error" {
			found = true
			hasMsg := false
			for _, f := range e.fields {
				if f.Key == "message" && f.Value != "" {
					hasMsg = true
				}
			}
			if !hasMsg {
				t.Errorf("error event missing message field, got %v", e.fields)
			}
		}
	}
	if !found {
		t.Error("expected error event when RunOnce fails in daemon mode")
	}
}

func TestRunMaxIter_ErrorEvent(t *testing.T) {
	h := &multiHarness{
		exitCodes: []int{1, 0},
		errs:      []error{errors.New("exit 1"), nil},
	}
	issue := beads.Issue{ID: "T-1", Title: "Fix"}
	b := &multiBeads{results: []beadsResult{
		{issues: []beads.Issue{issue}},
		{issues: []beads.Issue{issue}},
	}}
	l := &fakeLogger{}
	cfg := config.Config{
		Mode:          config.MaxIterationsMode,
		MaxIterations: 2,
		BeadsEnabled:  true,
		Prompt:        "do it",
	}

	err := RunMaxIter(context.Background(), cfg, h, b, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, e := range l.events {
		if e.name == "error" {
			found = true
			hasMsg := false
			for _, f := range e.fields {
				if f.Key == "message" && f.Value != "" {
					hasMsg = true
				}
			}
			if !hasMsg {
				t.Errorf("error event missing message field, got %v", e.fields)
			}
		}
	}
	if !found {
		t.Error("expected error event when iteration fails in max-iter mode")
	}
}

func TestRunOnce_InstructionPassedToPrompt(t *testing.T) {
	h := &fakeHarness{}
	b := &fakeBeads{issues: []beads.Issue{{ID: "TST-1", Title: "Fix bug"}}}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled:  true,
		BeadsInstruct: "custom instruction text",
		Prompt:        "do work",
	}

	ran, err := RunOnce(context.Background(), cfg, h, b, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Fatal("expected ranWork=true")
	}
	if !strings.Contains(h.prompt, "custom instruction text") {
		t.Errorf("expected prompt to contain instruction, got:\n%s", h.prompt)
	}
}

func TestRunOnce_DefaultInstructionWhenEmpty(t *testing.T) {
	h := &fakeHarness{}
	b := &fakeBeads{issues: []beads.Issue{{ID: "TST-1", Title: "Fix bug"}}}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled:  true,
		BeadsInstruct: "",
		Prompt:        "",
	}

	ran, err := RunOnce(context.Background(), cfg, h, b, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Fatal("expected ranWork=true")
	}
	if !strings.Contains(h.prompt, prompt.DefaultInstruction) {
		t.Errorf("expected default instruction in prompt, got:\n%s", h.prompt)
	}
}

func TestRunOnce_NoInstructionWithoutIssue(t *testing.T) {
	h := &fakeHarness{}
	l := &fakeLogger{}
	cfg := config.Config{
		BeadsEnabled:  false,
		BeadsInstruct: "should not appear",
		Prompt:        "just a prompt",
	}

	ran, err := RunOnce(context.Background(), cfg, h, nil, l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Fatal("expected ranWork=true")
	}
	if strings.Contains(h.prompt, "should not appear") {
		t.Errorf("instruction should not appear without issue, got:\n%s", h.prompt)
	}
}
