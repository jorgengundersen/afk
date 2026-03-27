package harness

import (
	"fmt"
	"io"
	"strings"
)

const maxInputDisplay = 200

// Renderer writes common events to a terminal-friendly text stream.
type Renderer struct {
	w io.Writer
}

// NewRenderer creates a Renderer that writes to w.
func NewRenderer(w io.Writer) *Renderer {
	return &Renderer{w: w}
}

// RenderStream reads common events from ch and renders each one as it arrives.
// It blocks until ch is closed.
func (r *Renderer) RenderStream(ch <-chan CommonEvent) {
	for ev := range ch {
		r.Render(ev)
	}
}

// Render writes a single common event to the output.
func (r *Renderer) Render(ev CommonEvent) {
	switch ev.Kind {
	case KindText:
		if ev.Text != nil && ev.Text.Content != "" {
			fmt.Fprintln(r.w, ev.Text.Content)
		}
	case KindToolCall:
		if ev.ToolCall != nil {
			input := truncate(ev.ToolCall.Input, maxInputDisplay)
			fmt.Fprintf(r.w, "[tool] %s: %s\n", ev.ToolCall.Name, input)
		}
	case KindToolOutput:
		if ev.ToolOutput != nil {
			content := truncate(ev.ToolOutput.Content, maxInputDisplay)
			fmt.Fprintf(r.w, "[result] %s\n", content)
		}
	case KindSummary:
		if ev.Summary != nil {
			r.renderSummary(ev.Summary)
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (r *Renderer) renderSummary(s *SummaryPayload) {
	if s.IsError {
		r.renderErrorSummary(s)
		return
	}

	var parts []string
	if s.DurationMS > 0 {
		parts = append(parts, fmt.Sprintf("%.1fs", float64(s.DurationMS)/1000.0))
	}
	if s.CostUSD > 0 {
		parts = append(parts, fmt.Sprintf("$%.4f", s.CostUSD))
	}
	if s.InputTokens > 0 || s.OutputTokens > 0 {
		parts = append(parts, fmt.Sprintf("%d tokens in / %d tokens out", s.InputTokens, s.OutputTokens))
	}
	if len(parts) == 0 {
		fmt.Fprintln(r.w, "[done]")
	} else {
		fmt.Fprintf(r.w, "[done] %s\n", strings.Join(parts, ", "))
	}
}

func (r *Renderer) renderErrorSummary(s *SummaryPayload) {
	var parts []string
	if s.DurationMS > 0 {
		parts = append(parts, fmt.Sprintf("%.1fs", float64(s.DurationMS)/1000.0))
	}
	if s.CostUSD > 0 {
		parts = append(parts, fmt.Sprintf("$%.4f", s.CostUSD))
	}
	if len(parts) > 0 {
		fmt.Fprintf(r.w, "[error] %s (%s)\n", s.Result, strings.Join(parts, ", "))
	} else {
		fmt.Fprintf(r.w, "[error] %s\n", s.Result)
	}
}
