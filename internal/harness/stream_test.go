package harness

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestParseStreamAssistantText(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello!"}]}}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Type != EventAssistant {
		t.Fatalf("expected type %q, got %q", EventAssistant, ev.Type)
	}
	if ev.Message == nil {
		t.Fatal("expected non-nil Message")
	}
	if len(ev.Message.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(ev.Message.Content))
	}
	if ev.Message.Content[0].Type != "text" {
		t.Fatalf("expected content type %q, got %q", "text", ev.Message.Content[0].Type)
	}
	if ev.Message.Content[0].Text != "Hello!" {
		t.Fatalf("expected text %q, got %q", "Hello!", ev.Message.Content[0].Text)
	}
}

func TestParseStreamAssistantToolUse(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Read","input":{"path":"/tmp/x"}}]}}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	cb := events[0].Message.Content[0]
	if cb.Type != "tool_use" {
		t.Fatalf("expected content type %q, got %q", "tool_use", cb.Type)
	}
	if cb.ID != "tu_1" {
		t.Fatalf("expected ID %q, got %q", "tu_1", cb.ID)
	}
	if cb.Name != "Read" {
		t.Fatalf("expected Name %q, got %q", "Read", cb.Name)
	}
	if string(cb.Input) != `{"path":"/tmp/x"}` {
		t.Fatalf("expected Input %q, got %q", `{"path":"/tmp/x"}`, string(cb.Input))
	}
}

func TestParseStreamToolResult(t *testing.T) {
	input := `{"type":"tool_result","tool_result":{"tool_use_id":"tu_1","content":"file contents here"}}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Type != EventToolResult {
		t.Fatalf("expected type %q, got %q", EventToolResult, ev.Type)
	}
	if ev.ToolResult.ToolUseID != "tu_1" {
		t.Fatalf("expected ToolUseID %q, got %q", "tu_1", ev.ToolResult.ToolUseID)
	}
	if ev.ToolResult.Content != "file contents here" {
		t.Fatalf("expected Content %q, got %q", "file contents here", ev.ToolResult.Content)
	}
}

func TestParseStreamResult(t *testing.T) {
	input := `{"type":"result","subtype":"success","cost_usd":0.003,"is_error":false,"duration_ms":1234,"result":"Done!"}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Type != EventResult {
		t.Fatalf("expected type %q, got %q", EventResult, ev.Type)
	}
	if ev.Result.CostUSD != 0.003 {
		t.Fatalf("expected CostUSD %f, got %f", 0.003, ev.Result.CostUSD)
	}
	if ev.Result.DurationMS != 1234 {
		t.Fatalf("expected DurationMS %d, got %d", 1234, ev.Result.DurationMS)
	}
	if ev.Result.Result != "Done!" {
		t.Fatalf("expected Result %q, got %q", "Done!", ev.Result.Result)
	}
	if ev.Result.IsError {
		t.Fatal("expected IsError false")
	}
}

func TestParseStreamSkipsMalformedJSON(t *testing.T) {
	input := "not json\n" + `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"ok"}]}}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event (malformed line skipped), got %d", len(events))
	}
	if events[0].Message.Content[0].Text != "ok" {
		t.Fatalf("expected text %q, got %q", "ok", events[0].Message.Content[0].Text)
	}
}

func TestParseStreamSkipsUnknownEventType(t *testing.T) {
	input := `{"type":"system","data":"something"}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event (unknown type skipped), got %d", len(events))
	}
	if events[0].Type != EventAssistant {
		t.Fatalf("expected type %q, got %q", EventAssistant, events[0].Type)
	}
}

func TestParseStreamSkipsEmptyLines(t *testing.T) {
	input := "\n\n" + `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}` + "\n\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestParseStreamMultipleEvents(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"thinking..."}]}}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"ls"}}]}}` + "\n" +
		`{"type":"tool_result","tool_result":{"tool_use_id":"tu_1","content":"file1.go\nfile2.go"}}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Found 2 files."}]}}` + "\n" +
		`{"type":"result","cost_usd":0.01,"duration_ms":5000,"result":"Found 2 files.","is_error":false}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
	wantTypes := []EventType{EventAssistant, EventAssistant, EventToolResult, EventAssistant, EventResult}
	for i, want := range wantTypes {
		if events[i].Type != want {
			t.Fatalf("event[%d]: expected type %q, got %q", i, want, events[i].Type)
		}
	}
}

func TestParseStreamContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Simulate: writer sends one event, then context is cancelled, then pipe closes
	// (mirrors real flow: subprocess killed → pw.Close())
	pr, pw := io.Pipe()
	ch := ParseStream(ctx, pr)

	// Write one valid event
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"before cancel"}]}}` + "\n"
	if _, err := pw.Write([]byte(line)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read the event
	ev, ok := <-ch
	if !ok {
		t.Fatal("expected event before cancel")
	}
	if ev.Message.Content[0].Text != "before cancel" {
		t.Fatalf("expected %q, got %q", "before cancel", ev.Message.Content[0].Text)
	}

	// Cancel context then close writer (simulates subprocess termination)
	cancel()
	pw.Close()

	// Channel should drain and close
	var extra []Event
	for ev := range ch {
		extra = append(extra, ev)
	}
	// No strict assertion on extra count — cancellation is prompt but not instant
}

func TestParseStreamResultWithError(t *testing.T) {
	input := `{"type":"result","subtype":"error","cost_usd":0.001,"is_error":true,"duration_ms":500,"result":"Something went wrong"}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if !events[0].Result.IsError {
		t.Fatal("expected IsError true")
	}
	if events[0].Result.Result != "Something went wrong" {
		t.Fatalf("expected Result %q, got %q", "Something went wrong", events[0].Result.Result)
	}
}

func TestParseStreamMixedContentBlocks(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Let me check."},{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"ls"}}]}}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if len(events[0].Message.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(events[0].Message.Content))
	}
	if events[0].Message.Content[0].Type != "text" {
		t.Fatalf("expected first block type %q, got %q", "text", events[0].Message.Content[0].Type)
	}
	if events[0].Message.Content[1].Type != "tool_use" {
		t.Fatalf("expected second block type %q, got %q", "tool_use", events[0].Message.Content[1].Type)
	}
}

// collectEvents drains ParseStream into a slice.
func collectEvents(t *testing.T, ctx context.Context, r *strings.Reader) []Event {
	t.Helper()
	ch := ParseStream(ctx, r)
	var events []Event
	for ev := range ch {
		events = append(events, ev)
	}
	return events
}
