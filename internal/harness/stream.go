package harness

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// claudeEventType represents the type of a Claude stream event (wire format).
type claudeEventType string

const (
	claudeAssistant  claudeEventType = "assistant"
	claudeToolResult claudeEventType = "tool_result"
	claudeResult     claudeEventType = "result"
)

// claudeMessage contains the content of an assistant turn (wire format).
type claudeMessage struct {
	Content []claudeContentBlock `json:"content"`
}

// claudeContentBlock is a piece of assistant content (wire format).
type claudeContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// claudeToolResultPayload contains the result of a tool invocation (wire format).
type claudeToolResultPayload struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

// rawEvent is the top-level JSON envelope from Claude's stream.
type rawEvent struct {
	Type       claudeEventType `json:"type"`
	Message    json.RawMessage `json:"message,omitempty"`
	ToolResult json.RawMessage `json:"tool_result,omitempty"`
	// Result-level fields are inlined at the top level.
	CostUSD    float64 `json:"cost_usd,omitempty"`
	DurationMS int     `json:"duration_ms,omitempty"`
	Result     string  `json:"result,omitempty"`
	IsError    bool    `json:"is_error,omitempty"`
}

// ParseStream reads newline-delimited JSON from r and sends parsed common
// events to the returned channel. The channel is closed when r is exhausted or
// ctx is cancelled. Malformed lines and unknown event types are skipped.
// If warn is non-nil, a message is written for each skipped line.
func ParseStream(ctx context.Context, r io.Reader, warn ...io.Writer) <-chan CommonEvent {
	var w io.Writer
	if len(warn) > 0 {
		w = warn[0]
	}

	ch := make(chan CommonEvent)
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

			events, ok := parseRawEvent(raw)
			if !ok {
				if w != nil {
					fmt.Fprintf(w, "skipping unknown event type: %s\n", raw.Type)
				}
				continue
			}

			for _, ev := range events {
				select {
				case ch <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}

func parseRawEvent(raw rawEvent) ([]CommonEvent, bool) {
	switch raw.Type {
	case claudeAssistant:
		var msg claudeMessage
		if err := json.Unmarshal(raw.Message, &msg); err != nil {
			return nil, false
		}
		var events []CommonEvent
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				events = append(events, CommonEvent{
					Kind: KindText,
					Text: &TextPayload{Content: block.Text},
				})
			case "tool_use":
				events = append(events, CommonEvent{
					Kind: KindToolCall,
					ToolCall: &ToolCallPayload{
						Name:  block.Name,
						Input: string(block.Input),
					},
				})
			}
		}
		if len(events) == 0 {
			return nil, false
		}
		return events, true
	case claudeToolResult:
		var tr claudeToolResultPayload
		if err := json.Unmarshal(raw.ToolResult, &tr); err != nil {
			return nil, false
		}
		return []CommonEvent{{
			Kind:       KindToolOutput,
			ToolOutput: &ToolOutputPayload{Content: tr.Content},
		}}, true
	case claudeResult:
		return []CommonEvent{{
			Kind: KindSummary,
			Summary: &SummaryPayload{
				CostUSD:    raw.CostUSD,
				DurationMS: raw.DurationMS,
				Result:     raw.Result,
				IsError:    raw.IsError,
			},
		}}, true
	default:
		return nil, false
	}
}
