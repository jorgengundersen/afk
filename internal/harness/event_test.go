package harness

import "testing"

func TestEventKindConstants(t *testing.T) {
	tests := []struct {
		kind EventKind
		want string
	}{
		{KindText, "text"},
		{KindToolCall, "tool_call"},
		{KindToolOutput, "tool_output"},
		{KindSummary, "summary"},
	}
	for _, tt := range tests {
		if string(tt.kind) != tt.want {
			t.Errorf("got %q, want %q", tt.kind, tt.want)
		}
	}
}

func TestCommonEventPayloads(t *testing.T) {
	t.Run("text event", func(t *testing.T) {
		ev := CommonEvent{
			Kind: KindText,
			Text: &TextPayload{Content: "hello"},
		}
		if ev.Kind != KindText {
			t.Fatalf("kind = %q, want %q", ev.Kind, KindText)
		}
		if ev.Text.Content != "hello" {
			t.Fatalf("text content = %q, want %q", ev.Text.Content, "hello")
		}
	})

	t.Run("tool call event", func(t *testing.T) {
		ev := CommonEvent{
			Kind:     KindToolCall,
			ToolCall: &ToolCallPayload{Name: "Edit", Input: "file.go"},
		}
		if ev.ToolCall.Name != "Edit" {
			t.Fatalf("tool name = %q, want %q", ev.ToolCall.Name, "Edit")
		}
		if ev.ToolCall.Input != "file.go" {
			t.Fatalf("tool input = %q, want %q", ev.ToolCall.Input, "file.go")
		}
	})

	t.Run("tool output event", func(t *testing.T) {
		ev := CommonEvent{
			Kind:       KindToolOutput,
			ToolOutput: &ToolOutputPayload{Content: "done"},
		}
		if ev.ToolOutput.Content != "done" {
			t.Fatalf("tool output = %q, want %q", ev.ToolOutput.Content, "done")
		}
	})

	t.Run("summary event with all fields", func(t *testing.T) {
		ev := CommonEvent{
			Kind: KindSummary,
			Summary: &SummaryPayload{
				DurationMS:   12300,
				CostUSD:      0.0412,
				InputTokens:  1234,
				OutputTokens: 567,
				Result:       "completed",
				IsError:      false,
			},
		}
		s := ev.Summary
		if s.DurationMS != 12300 {
			t.Fatalf("duration = %d, want %d", s.DurationMS, 12300)
		}
		if s.CostUSD != 0.0412 {
			t.Fatalf("cost = %f, want %f", s.CostUSD, 0.0412)
		}
		if s.InputTokens != 1234 {
			t.Fatalf("input tokens = %d, want %d", s.InputTokens, 1234)
		}
		if s.OutputTokens != 567 {
			t.Fatalf("output tokens = %d, want %d", s.OutputTokens, 567)
		}
		if s.Result != "completed" {
			t.Fatalf("result = %q, want %q", s.Result, "completed")
		}
		if s.IsError {
			t.Fatal("is_error = true, want false")
		}
	})

	t.Run("summary zero values when unavailable", func(t *testing.T) {
		ev := CommonEvent{
			Kind:    KindSummary,
			Summary: &SummaryPayload{},
		}
		s := ev.Summary
		if s.DurationMS != 0 || s.CostUSD != 0 || s.InputTokens != 0 || s.OutputTokens != 0 {
			t.Fatal("zero-valued summary fields should default to zero")
		}
		if s.Result != "" {
			t.Fatalf("result = %q, want empty", s.Result)
		}
		if s.IsError {
			t.Fatal("is_error should default to false")
		}
	})
}
