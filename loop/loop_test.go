package loop

import (
	"context"
	"errors"
	"testing"

	"github.com/jorgengundersen/afk/beads"
	"github.com/jorgengundersen/afk/config"
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
	if l.events[0].name != "iteration_start" {
		t.Errorf("first event = %q, want iteration_start", l.events[0].name)
	}
	last := l.events[len(l.events)-1]
	if last.name != "iteration_end" {
		t.Errorf("last event = %q, want iteration_end", last.name)
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

	// Check loop_start and loop_end events
	if l.events[0].name != "loop_start" {
		t.Errorf("first event = %q, want loop_start", l.events[0].name)
	}
	last := l.events[len(l.events)-1]
	if last.name != "loop_end" {
		t.Errorf("last event = %q, want loop_end", last.name)
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
	if last.name != "loop_end" {
		t.Errorf("last event = %q, want loop_end", last.name)
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

func hasField(fields []Field, key, value string) bool {
	for _, f := range fields {
		if f.Key == key && f.Value == value {
			return true
		}
	}
	return false
}
