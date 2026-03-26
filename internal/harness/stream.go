package harness

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// EventType represents the type of a Claude stream event.
type EventType string

const (
	EventAssistant  EventType = "assistant"
	EventToolResult EventType = "tool_result"
	EventResult     EventType = "result"
)

// Event represents a parsed Claude stream event.
type Event struct {
	Type       EventType
	Message    *AssistantMessage // non-nil for EventAssistant
	ToolResult *ToolResult       // non-nil for EventToolResult
	Result     *ResultSummary    // non-nil for EventResult
}

// AssistantMessage contains the content of an assistant turn.
type AssistantMessage struct {
	Content []ContentBlock `json:"content"`
}

// ContentBlock is a piece of assistant content.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ToolResult contains the result of a tool invocation.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

// ResultSummary contains the final summary of a Claude session.
type ResultSummary struct {
	CostUSD    float64 `json:"cost_usd"`
	DurationMS int     `json:"duration_ms"`
	Result     string  `json:"result"`
	IsError    bool    `json:"is_error"`
}

// rawEvent is the top-level JSON envelope from Claude's stream.
type rawEvent struct {
	Type       EventType       `json:"type"`
	Message    json.RawMessage `json:"message,omitempty"`
	ToolResult json.RawMessage `json:"tool_result,omitempty"`
	// Result-level fields are inlined at the top level.
	CostUSD    float64 `json:"cost_usd,omitempty"`
	DurationMS int     `json:"duration_ms,omitempty"`
	Result     string  `json:"result,omitempty"`
	IsError    bool    `json:"is_error,omitempty"`
}

// ParseStream reads newline-delimited JSON from r and sends parsed events
// to the returned channel. The channel is closed when r is exhausted or
// ctx is cancelled. Malformed lines and unknown event types are skipped.
// If warn is non-nil, a message is written for each skipped line.
func ParseStream(ctx context.Context, r io.Reader, warn ...io.Writer) <-chan Event {
	var w io.Writer
	if len(warn) > 0 {
		w = warn[0]
	}

	ch := make(chan Event)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var raw rawEvent
			if err := json.Unmarshal(line, &raw); err != nil {
				if w != nil {
					fmt.Fprintf(w, "skipping malformed JSON: %s\n", line)
				}
				continue
			}

			ev, ok := parseRawEvent(raw)
			if !ok {
				if w != nil {
					fmt.Fprintf(w, "skipping unknown event type: %s\n", raw.Type)
				}
				continue
			}

			select {
			case ch <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

func parseRawEvent(raw rawEvent) (Event, bool) {
	switch raw.Type {
	case EventAssistant:
		var msg AssistantMessage
		if err := json.Unmarshal(raw.Message, &msg); err != nil {
			return Event{}, false
		}
		return Event{Type: EventAssistant, Message: &msg}, true
	case EventToolResult:
		var tr ToolResult
		if err := json.Unmarshal(raw.ToolResult, &tr); err != nil {
			return Event{}, false
		}
		return Event{Type: EventToolResult, ToolResult: &tr}, true
	case EventResult:
		return Event{
			Type: EventResult,
			Result: &ResultSummary{
				CostUSD:    raw.CostUSD,
				DurationMS: raw.DurationMS,
				Result:     raw.Result,
				IsError:    raw.IsError,
			},
		}, true
	default:
		return Event{}, false
	}
}
