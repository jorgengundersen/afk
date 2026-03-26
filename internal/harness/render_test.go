package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestRenderTextEvent(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := Event{
		Type: EventAssistant,
		Message: &AssistantMessage{
			Content: []ContentBlock{
				{Type: "text", Text: "Hello, world!"},
			},
		},
	}
	r.Render(ev)

	got := buf.String()
	want := "Hello, world!\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRenderToolUseEvent(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := Event{
		Type: EventAssistant,
		Message: &AssistantMessage{
			Content: []ContentBlock{
				{Type: "tool_use", Name: "Bash", Input: json.RawMessage(`{"command":"ls -la"}`)},
			},
		},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "Bash") {
		t.Fatalf("expected tool name in output, got %q", got)
	}
	if !strings.Contains(got, "ls -la") {
		t.Fatalf("expected tool input in output, got %q", got)
	}
}

func TestRenderToolResultEvent(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := Event{
		Type: EventToolResult,
		ToolResult: &ToolResult{
			ToolUseID: "tu_1",
			Content:   "file1.go\nfile2.go",
		},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "file1.go") {
		t.Fatalf("expected tool result content in output, got %q", got)
	}
}

func TestRenderToolResultTruncated(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	long := strings.Repeat("x", 500)
	ev := Event{
		Type: EventToolResult,
		ToolResult: &ToolResult{
			ToolUseID: "tu_1",
			Content:   long,
		},
	}
	r.Render(ev)

	got := buf.String()
	if len(got) >= 500 {
		t.Fatalf("expected truncated output, got length %d", len(got))
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("expected truncation marker, got %q", got)
	}
}

func TestRenderResultSummary(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := Event{
		Type: EventResult,
		Result: &ResultSummary{
			CostUSD:    0.003,
			DurationMS: 1234,
			Result:     "Done!",
			IsError:    false,
		},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "1.2s") {
		t.Fatalf("expected duration in output, got %q", got)
	}
	if !strings.Contains(got, "$0.0030") {
		t.Fatalf("expected cost in output, got %q", got)
	}
}

func TestRenderResultSummaryError(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := Event{
		Type: EventResult,
		Result: &ResultSummary{
			CostUSD:    0.001,
			DurationMS: 500,
			Result:     "Something went wrong",
			IsError:    true,
		},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "error") && !strings.Contains(got, "Error") && !strings.Contains(got, "ERROR") {
		t.Fatalf("expected error indicator in output, got %q", got)
	}
	if !strings.Contains(got, "Something went wrong") {
		t.Fatalf("expected error message in output, got %q", got)
	}
}

func TestRenderTextWithNewlines(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := Event{
		Type: EventAssistant,
		Message: &AssistantMessage{
			Content: []ContentBlock{
				{Type: "text", Text: "Line 1\nLine 2\nLine 3"},
			},
		},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "Line 1\nLine 2\nLine 3") {
		t.Fatalf("expected preserved newlines, got %q", got)
	}
}

func TestRenderMixedContentBlocks(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := Event{
		Type: EventAssistant,
		Message: &AssistantMessage{
			Content: []ContentBlock{
				{Type: "text", Text: "Let me check."},
				{Type: "tool_use", Name: "Bash", Input: json.RawMessage(`{"command":"ls"}`)},
			},
		},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "Let me check.") {
		t.Fatalf("expected text in output, got %q", got)
	}
	if !strings.Contains(got, "Bash") {
		t.Fatalf("expected tool name in output, got %q", got)
	}
}

func TestRenderToolUseInputTruncated(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	longInput := `{"command":"` + strings.Repeat("a", 300) + `"}`
	ev := Event{
		Type: EventAssistant,
		Message: &AssistantMessage{
			Content: []ContentBlock{
				{Type: "tool_use", Name: "Bash", Input: json.RawMessage(longInput)},
			},
		},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "...") {
		t.Fatalf("expected truncation marker, got %q", got)
	}
}

func TestRenderStreamIntegration(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"thinking..."}]}}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"ls"}}]}}` + "\n" +
		`{"type":"tool_result","tool_result":{"tool_use_id":"tu_1","content":"file1.go"}}` + "\n" +
		`{"type":"result","cost_usd":0.01,"duration_ms":5000,"result":"Done","is_error":false}` + "\n"

	var buf bytes.Buffer
	r := NewRenderer(&buf)
	r.RenderStream(context.Background(), strings.NewReader(input))

	got := buf.String()
	if !strings.Contains(got, "thinking...") {
		t.Fatalf("expected agent text, got %q", got)
	}
	if !strings.Contains(got, "[tool] Bash") {
		t.Fatalf("expected tool use, got %q", got)
	}
	if !strings.Contains(got, "[result]") {
		t.Fatalf("expected tool result, got %q", got)
	}
	if !strings.Contains(got, "[done]") {
		t.Fatalf("expected done summary, got %q", got)
	}
}

func TestRenderStreamStopsOnPipeClose(t *testing.T) {
	pr, pw := io.Pipe()

	var buf bytes.Buffer
	r := NewRenderer(&buf)

	done := make(chan struct{})
	go func() {
		r.RenderStream(context.Background(), pr)
		close(done)
	}()

	// Write one event then close pipe (simulates subprocess exit)
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]}}` + "\n"
	pw.Write([]byte(line))
	pw.Close()

	<-done

	got := buf.String()
	if !strings.Contains(got, "hello") {
		t.Fatalf("expected rendered event, got %q", got)
	}
}
