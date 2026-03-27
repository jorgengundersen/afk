package harness

import (
	"bytes"
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
	if ev.Kind != KindText {
		t.Fatalf("expected kind %q, got %q", KindText, ev.Kind)
	}
	if ev.Text == nil {
		t.Fatal("expected non-nil Text payload")
	}
	if ev.Text.Content != "Hello!" {
		t.Fatalf("expected text %q, got %q", "Hello!", ev.Text.Content)
	}
}

func TestParseStreamAssistantToolUse(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Read","input":{"path":"/tmp/x"}}]}}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Kind != KindToolCall {
		t.Fatalf("expected kind %q, got %q", KindToolCall, ev.Kind)
	}
	if ev.ToolCall == nil {
		t.Fatal("expected non-nil ToolCall payload")
	}
	if ev.ToolCall.Name != "Read" {
		t.Fatalf("expected Name %q, got %q", "Read", ev.ToolCall.Name)
	}
	if ev.ToolCall.Input != `{"path":"/tmp/x"}` {
		t.Fatalf("expected Input %q, got %q", `{"path":"/tmp/x"}`, ev.ToolCall.Input)
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
	if ev.Kind != KindToolOutput {
		t.Fatalf("expected kind %q, got %q", KindToolOutput, ev.Kind)
	}
	if ev.ToolOutput == nil {
		t.Fatal("expected non-nil ToolOutput payload")
	}
	if ev.ToolOutput.Content != "file contents here" {
		t.Fatalf("expected Content %q, got %q", "file contents here", ev.ToolOutput.Content)
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
	if ev.Kind != KindSummary {
		t.Fatalf("expected kind %q, got %q", KindSummary, ev.Kind)
	}
	if ev.Summary == nil {
		t.Fatal("expected non-nil Summary payload")
	}
	if ev.Summary.CostUSD != 0.003 {
		t.Fatalf("expected CostUSD %f, got %f", 0.003, ev.Summary.CostUSD)
	}
	if ev.Summary.DurationMS != 1234 {
		t.Fatalf("expected DurationMS %d, got %d", 1234, ev.Summary.DurationMS)
	}
	if ev.Summary.Result != "Done!" {
		t.Fatalf("expected Result %q, got %q", "Done!", ev.Summary.Result)
	}
	if ev.Summary.IsError {
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
	if events[0].Text.Content != "ok" {
		t.Fatalf("expected text %q, got %q", "ok", events[0].Text.Content)
	}
}

func TestParseStreamWarnsMalformedJSON(t *testing.T) {
	input := "not json\n" + `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"ok"}]}}` + "\n"
	r := strings.NewReader(input)

	var warn bytes.Buffer
	ch := ParseStream(context.Background(), r, &warn)
	for range ch {
	}

	got := warn.String()
	if !strings.Contains(got, "skipping malformed JSON") {
		t.Errorf("expected malformed JSON warning, got %q", got)
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
	if events[0].Kind != KindText {
		t.Fatalf("expected kind %q, got %q", KindText, events[0].Kind)
	}
}

func TestParseStreamWarnsUnknownEventType(t *testing.T) {
	input := `{"type":"system","data":"something"}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}` + "\n"
	r := strings.NewReader(input)

	var warn bytes.Buffer
	ch := ParseStream(context.Background(), r, &warn)
	for range ch {
	}

	got := warn.String()
	if !strings.Contains(got, "skipping unknown event type") {
		t.Errorf("expected unknown event type warning, got %q", got)
	}
	if !strings.Contains(got, "system") {
		t.Errorf("expected warning to include event type name, got %q", got)
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
	wantKinds := []EventKind{KindText, KindToolCall, KindToolOutput, KindText, KindSummary}
	for i, want := range wantKinds {
		if events[i].Kind != want {
			t.Fatalf("event[%d]: expected kind %q, got %q", i, want, events[i].Kind)
		}
	}
}

func TestParseStreamContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

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
	if ev.Text.Content != "before cancel" {
		t.Fatalf("expected %q, got %q", "before cancel", ev.Text.Content)
	}

	// Cancel context then close writer
	cancel()
	pw.Close()

	// Channel should drain and close
	for range ch {
	}
}

func TestParseStreamResultWithError(t *testing.T) {
	input := `{"type":"result","subtype":"error","cost_usd":0.001,"is_error":true,"duration_ms":500,"result":"Something went wrong"}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if !events[0].Summary.IsError {
		t.Fatal("expected IsError true")
	}
	if events[0].Summary.Result != "Something went wrong" {
		t.Fatalf("expected Result %q, got %q", "Something went wrong", events[0].Summary.Result)
	}
}

func TestParseStreamMixedContentBlocks(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Let me check."},{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"ls"}}]}}` + "\n"
	r := strings.NewReader(input)

	events := collectEvents(t, context.Background(), r)

	// Mixed content blocks should emit separate events
	if len(events) != 2 {
		t.Fatalf("expected 2 events (one Text, one ToolCall), got %d", len(events))
	}
	if events[0].Kind != KindText {
		t.Fatalf("expected first event kind %q, got %q", KindText, events[0].Kind)
	}
	if events[0].Text.Content != "Let me check." {
		t.Fatalf("expected text %q, got %q", "Let me check.", events[0].Text.Content)
	}
	if events[1].Kind != KindToolCall {
		t.Fatalf("expected second event kind %q, got %q", KindToolCall, events[1].Kind)
	}
	if events[1].ToolCall.Name != "Bash" {
		t.Fatalf("expected tool name %q, got %q", "Bash", events[1].ToolCall.Name)
	}
}

// collectEvents drains ParseStream into a slice.
func collectEvents(t *testing.T, ctx context.Context, r *strings.Reader) []CommonEvent {
	t.Helper()
	ch := ParseStream(ctx, r)
	var events []CommonEvent
	for ev := range ch {
		events = append(events, ev)
	}
	return events
}
