package harness

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRenderTextEvent(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := CommonEvent{
		Kind: KindText,
		Text: &TextPayload{Content: "Hello, world!"},
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

	ev := CommonEvent{
		Kind:     KindToolCall,
		ToolCall: &ToolCallPayload{Name: "Bash", Input: `{"command":"ls -la"}`},
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

	ev := CommonEvent{
		Kind:       KindToolOutput,
		ToolOutput: &ToolOutputPayload{Content: "file1.go\nfile2.go"},
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
	ev := CommonEvent{
		Kind:       KindToolOutput,
		ToolOutput: &ToolOutputPayload{Content: long},
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

	ev := CommonEvent{
		Kind: KindSummary,
		Summary: &SummaryPayload{
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

	ev := CommonEvent{
		Kind: KindSummary,
		Summary: &SummaryPayload{
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

func TestRenderResultSummaryWithTokens(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := CommonEvent{
		Kind: KindSummary,
		Summary: &SummaryPayload{
			InputTokens:  1234,
			OutputTokens: 567,
		},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "1234") {
		t.Fatalf("expected input token count in output, got %q", got)
	}
	if !strings.Contains(got, "567") {
		t.Fatalf("expected output token count in output, got %q", got)
	}
}

func TestRenderResultSummaryWithAllFields(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := CommonEvent{
		Kind: KindSummary,
		Summary: &SummaryPayload{
			DurationMS:   8100,
			CostUSD:      0.0412,
			InputTokens:  1234,
			OutputTokens: 567,
		},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "8.1s") {
		t.Fatalf("expected duration in output, got %q", got)
	}
	if !strings.Contains(got, "$0.0412") {
		t.Fatalf("expected cost in output, got %q", got)
	}
	if !strings.Contains(got, "1234") {
		t.Fatalf("expected input token count in output, got %q", got)
	}
	if !strings.Contains(got, "567") {
		t.Fatalf("expected output token count in output, got %q", got)
	}
}

func TestRenderTextWithNewlines(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)

	ev := CommonEvent{
		Kind: KindText,
		Text: &TextPayload{Content: "Line 1\nLine 2\nLine 3"},
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

	// Mixed content blocks are now separate events
	r.Render(CommonEvent{Kind: KindText, Text: &TextPayload{Content: "Let me check."}})
	r.Render(CommonEvent{Kind: KindToolCall, ToolCall: &ToolCallPayload{Name: "Bash", Input: `{"command":"ls"}`}})

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
	ev := CommonEvent{
		Kind:     KindToolCall,
		ToolCall: &ToolCallPayload{Name: "Bash", Input: longInput},
	}
	r.Render(ev)

	got := buf.String()
	if !strings.Contains(got, "...") {
		t.Fatalf("expected truncation marker, got %q", got)
	}
}

func TestRenderStreamFromChannel(t *testing.T) {
	ch := make(chan CommonEvent, 4)
	ch <- CommonEvent{Kind: KindText, Text: &TextPayload{Content: "thinking..."}}
	ch <- CommonEvent{Kind: KindToolCall, ToolCall: &ToolCallPayload{Name: "Bash", Input: `{"command":"ls"}`}}
	ch <- CommonEvent{Kind: KindToolOutput, ToolOutput: &ToolOutputPayload{Content: "file1.go"}}
	ch <- CommonEvent{Kind: KindSummary, Summary: &SummaryPayload{CostUSD: 0.01, DurationMS: 5000}}
	close(ch)

	var buf bytes.Buffer
	r := NewRenderer(&buf)
	r.RenderStream(ch)

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

func TestRenderStreamClosedChannel(t *testing.T) {
	ch := make(chan CommonEvent)
	close(ch)

	var buf bytes.Buffer
	r := NewRenderer(&buf)
	r.RenderStream(ch)

	if buf.String() != "" {
		t.Fatalf("expected empty output for closed channel, got %q", buf.String())
	}
}

func TestRenderClaudeStreamIntegration(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"thinking..."}]}}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"ls"}}]}}` + "\n" +
		`{"type":"tool_result","tool_result":{"tool_use_id":"tu_1","content":"file1.go"}}` + "\n" +
		`{"type":"result","cost_usd":0.01,"duration_ms":5000,"result":"Done","is_error":false}` + "\n"

	ch := ParseStream(context.Background(), strings.NewReader(input))

	var buf bytes.Buffer
	r := NewRenderer(&buf)
	r.RenderStream(ch)

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
