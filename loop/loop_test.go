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

func hasField(fields []Field, key, value string) bool {
	for _, f := range fields {
		if f.Key == key && f.Value == value {
			return true
		}
	}
	return false
}
