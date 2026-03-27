package harness

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func collectCodexEvents(t *testing.T, input string) []CommonEvent {
	t.Helper()
	ch := ParseCodexStream(context.Background(), strings.NewReader(input))
	var events []CommonEvent
	for ev := range ch {
		events = append(events, ev)
	}
	return events
}

func TestCodexAgentMessage(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"agent_message","content":"Hello!"}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindText {
		t.Fatalf("expected kind %q, got %q", KindText, events[0].Kind)
	}
	if events[0].Text.Content != "Hello!" {
		t.Fatalf("expected text %q, got %q", "Hello!", events[0].Text.Content)
	}
}

func TestCodexReasoning(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"reasoning","content":"Let me think..."}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindText {
		t.Fatalf("expected kind %q, got %q", KindText, events[0].Kind)
	}
	if events[0].Text.Content != "Let me think..." {
		t.Fatalf("expected text %q, got %q", "Let me think...", events[0].Text.Content)
	}
}

func TestCodexCommandExecutionStarted(t *testing.T) {
	input := `{"type":"item.started","item":{"type":"command_execution","command":"ls -la"}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindToolCall {
		t.Fatalf("expected kind %q, got %q", KindToolCall, events[0].Kind)
	}
	if events[0].ToolCall.Name != "shell" {
		t.Fatalf("expected tool name %q, got %q", "shell", events[0].ToolCall.Name)
	}
	if events[0].ToolCall.Input != "ls -la" {
		t.Fatalf("expected input %q, got %q", "ls -la", events[0].ToolCall.Input)
	}
}

func TestCodexCommandExecutionCompleted(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"command_execution","command":"ls","aggregated_output":"file1\nfile2","exit_code":0}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindToolOutput {
		t.Fatalf("expected kind %q, got %q", KindToolOutput, events[0].Kind)
	}
	if events[0].ToolOutput.Content != "file1\nfile2" {
		t.Fatalf("expected content %q, got %q", "file1\nfile2", events[0].ToolOutput.Content)
	}
}

func TestCodexCommandExecutionWithExitCode(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"command_execution","command":"false","aggregated_output":"","exit_code":1}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if !strings.Contains(events[0].ToolOutput.Content, "exit code 1") {
		t.Fatalf("expected exit code in content, got %q", events[0].ToolOutput.Content)
	}
}

func TestCodexMCPToolCallStarted(t *testing.T) {
	input := `{"type":"item.started","item":{"type":"mcp_tool_call","tool_name":"read_file","arguments":"{\"path\":\"foo.go\"}"}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindToolCall {
		t.Fatalf("expected kind %q, got %q", KindToolCall, events[0].Kind)
	}
	if events[0].ToolCall.Name != "read_file" {
		t.Fatalf("expected tool name %q, got %q", "read_file", events[0].ToolCall.Name)
	}
	if events[0].ToolCall.Input != `{"path":"foo.go"}` {
		t.Fatalf("expected input %q, got %q", `{"path":"foo.go"}`, events[0].ToolCall.Input)
	}
}

func TestCodexMCPToolCallCompleted(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"mcp_tool_call","tool_name":"read_file","result":"package main"}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindToolOutput {
		t.Fatalf("expected kind %q, got %q", KindToolOutput, events[0].Kind)
	}
	if events[0].ToolOutput.Content != "package main" {
		t.Fatalf("expected content %q, got %q", "package main", events[0].ToolOutput.Content)
	}
}

func TestCodexFileChange(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"file_change","changes":[{"path":"foo.go","kind":"modified"},{"path":"bar.go","kind":"created"}]}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindToolOutput {
		t.Fatalf("expected kind %q, got %q", KindToolOutput, events[0].Kind)
	}
	if !strings.Contains(events[0].ToolOutput.Content, "foo.go") {
		t.Fatalf("expected foo.go in content, got %q", events[0].ToolOutput.Content)
	}
	if !strings.Contains(events[0].ToolOutput.Content, "bar.go") {
		t.Fatalf("expected bar.go in content, got %q", events[0].ToolOutput.Content)
	}
}

func TestCodexWebSearchStarted(t *testing.T) {
	input := `{"type":"item.started","item":{"type":"web_search","query":"golang testing"}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindToolCall {
		t.Fatalf("expected kind %q, got %q", KindToolCall, events[0].Kind)
	}
	if events[0].ToolCall.Name != "web_search" {
		t.Fatalf("expected tool name %q, got %q", "web_search", events[0].ToolCall.Name)
	}
	if events[0].ToolCall.Input != "golang testing" {
		t.Fatalf("expected input %q, got %q", "golang testing", events[0].ToolCall.Input)
	}
}

func TestCodexTurnCompleted(t *testing.T) {
	input := `{"type":"turn.completed","usage":{"input_tokens":1234,"output_tokens":567}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindSummary {
		t.Fatalf("expected kind %q, got %q", KindSummary, events[0].Kind)
	}
	if events[0].Summary.InputTokens != 1234 {
		t.Fatalf("expected input tokens %d, got %d", 1234, events[0].Summary.InputTokens)
	}
	if events[0].Summary.OutputTokens != 567 {
		t.Fatalf("expected output tokens %d, got %d", 567, events[0].Summary.OutputTokens)
	}
	if events[0].Summary.IsError {
		t.Fatal("expected IsError false")
	}
}

func TestCodexTurnFailed(t *testing.T) {
	input := `{"type":"turn.failed","error":"something went wrong"}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != KindSummary {
		t.Fatalf("expected kind %q, got %q", KindSummary, events[0].Kind)
	}
	if !events[0].Summary.IsError {
		t.Fatal("expected IsError true")
	}
	if events[0].Summary.Result != "something went wrong" {
		t.Fatalf("expected result %q, got %q", "something went wrong", events[0].Summary.Result)
	}
}

func TestCodexSkipsInformationalEvents(t *testing.T) {
	input := `{"type":"thread.started"}` + "\n" +
		`{"type":"turn.started"}` + "\n" +
		`{"type":"item.completed","item":{"type":"todo_list","items":["task1"]}}` + "\n" +
		`{"type":"item.completed","item":{"type":"agent_message","content":"hello"}}` + "\n"
	events := collectCodexEvents(t, input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event (informational skipped), got %d", len(events))
	}
	if events[0].Kind != KindText {
		t.Fatalf("expected kind %q, got %q", KindText, events[0].Kind)
	}
}

func TestCodexSkipsMalformedJSON(t *testing.T) {
	input := "not json\n" + `{"type":"item.completed","item":{"type":"agent_message","content":"ok"}}` + "\n"

	var warn bytes.Buffer
	ch := ParseCodexStream(context.Background(), strings.NewReader(input), &warn)
	var events []CommonEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if !strings.Contains(warn.String(), "skipping malformed JSON") {
		t.Fatalf("expected malformed JSON warning, got %q", warn.String())
	}
}

func TestCodexMultipleEvents(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"reasoning","content":"thinking"}}` + "\n" +
		`{"type":"item.started","item":{"type":"command_execution","command":"ls"}}` + "\n" +
		`{"type":"item.completed","item":{"type":"command_execution","command":"ls","aggregated_output":"file.go","exit_code":0}}` + "\n" +
		`{"type":"item.completed","item":{"type":"agent_message","content":"Found it."}}` + "\n" +
		`{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":50}}` + "\n"
	events := collectCodexEvents(t, input)

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

func TestCodexContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch := ParseCodexStream(ctx, strings.NewReader(`{"type":"item.completed","item":{"type":"agent_message","content":"hi"}}`+"\n"))
	var events []CommonEvent
	for ev := range ch {
		events = append(events, ev)
	}
	// Channel should close promptly; no strict assertion on count
}
