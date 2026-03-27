package harness

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// codexEvent is the top-level JSON envelope from Codex's NDJSON stream.
type codexEvent struct {
	Type  string          `json:"type"`
	Item  json.RawMessage `json:"item,omitempty"`
	Usage *codexUsage     `json:"usage,omitempty"`
	Error string          `json:"error,omitempty"`
}

// codexItem is a Codex stream item with a type discriminator.
type codexItem struct {
	Type             string        `json:"type"`
	Content          string        `json:"content,omitempty"`
	Command          string        `json:"command,omitempty"`
	AggregatedOutput string        `json:"aggregated_output,omitempty"`
	ExitCode         int           `json:"exit_code,omitempty"`
	ToolName         string        `json:"tool_name,omitempty"`
	Arguments        string        `json:"arguments,omitempty"`
	Result           string        `json:"result,omitempty"`
	Query            string        `json:"query,omitempty"`
	Changes          []codexChange `json:"changes,omitempty"`
}

// codexChange represents a file change in a Codex file_change event.
type codexChange struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

// codexUsage contains token usage from a Codex turn.
type codexUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ParseCodexStream reads newline-delimited JSON from r (Codex exec --json)
// and sends parsed common events to the returned channel. The channel is
// closed when r is exhausted or ctx is cancelled.
func ParseCodexStream(ctx context.Context, r io.Reader, warn ...io.Writer) <-chan CommonEvent {
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

			var raw codexEvent
			if err := json.Unmarshal(line, &raw); err != nil {
				if w != nil {
					fmt.Fprintf(w, "skipping malformed JSON: %s\n", line)
				}
				continue
			}

			events := parseCodexEvent(raw, w)
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

func parseCodexEvent(raw codexEvent, w io.Writer) []CommonEvent {
	switch raw.Type {
	case "item.started":
		return parseCodexItemStarted(raw.Item, w)
	case "item.completed":
		return parseCodexItemCompleted(raw.Item, w)
	case "turn.completed":
		if raw.Usage != nil {
			return []CommonEvent{{
				Kind: KindSummary,
				Summary: &SummaryPayload{
					InputTokens:  raw.Usage.InputTokens,
					OutputTokens: raw.Usage.OutputTokens,
				},
			}}
		}
		return nil
	case "turn.failed":
		return []CommonEvent{{
			Kind: KindSummary,
			Summary: &SummaryPayload{
				IsError: true,
				Result:  raw.Error,
			},
		}}
	default:
		// Informational events (thread.started, turn.started, etc.) are skipped
		return nil
	}
}

func parseCodexItemStarted(data json.RawMessage, w io.Writer) []CommonEvent {
	if data == nil {
		return nil
	}
	var item codexItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil
	}

	switch item.Type {
	case "command_execution":
		return []CommonEvent{{
			Kind:     KindToolCall,
			ToolCall: &ToolCallPayload{Name: "shell", Input: item.Command},
		}}
	case "mcp_tool_call":
		return []CommonEvent{{
			Kind:     KindToolCall,
			ToolCall: &ToolCallPayload{Name: item.ToolName, Input: item.Arguments},
		}}
	case "web_search":
		return []CommonEvent{{
			Kind:     KindToolCall,
			ToolCall: &ToolCallPayload{Name: "web_search", Input: item.Query},
		}}
	default:
		return nil
	}
}

func parseCodexItemCompleted(data json.RawMessage, w io.Writer) []CommonEvent {
	if data == nil {
		return nil
	}
	var item codexItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil
	}

	switch item.Type {
	case "agent_message", "reasoning":
		return []CommonEvent{{
			Kind: KindText,
			Text: &TextPayload{Content: item.Content},
		}}
	case "command_execution":
		content := item.AggregatedOutput
		if item.ExitCode != 0 {
			if content != "" {
				content += fmt.Sprintf(" (exit code %d)", item.ExitCode)
			} else {
				content = fmt.Sprintf("exit code %d", item.ExitCode)
			}
		}
		return []CommonEvent{{
			Kind:       KindToolOutput,
			ToolOutput: &ToolOutputPayload{Content: content},
		}}
	case "mcp_tool_call":
		return []CommonEvent{{
			Kind:       KindToolOutput,
			ToolOutput: &ToolOutputPayload{Content: item.Result},
		}}
	case "file_change":
		var parts []string
		for _, c := range item.Changes {
			parts = append(parts, fmt.Sprintf("%s (%s)", c.Path, c.Kind))
		}
		return []CommonEvent{{
			Kind:       KindToolOutput,
			ToolOutput: &ToolOutputPayload{Content: strings.Join(parts, ", ")},
		}}
	default:
		// Unknown item types (todo_list, collab_tool_call, etc.) are skipped
		return nil
	}
}
