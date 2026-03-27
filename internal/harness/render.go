package harness

import (
	"context"
	"fmt"
	"io"
	"os"
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

// RenderStream parses a Claude JSON stream and renders each event as it arrives.
func (r *Renderer) RenderStream(ctx context.Context, reader io.Reader) {
	for ev := range ParseStream(ctx, reader, os.Stderr) {
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
	duration := float64(s.DurationMS) / 1000.0
	if s.IsError {
		fmt.Fprintf(r.w, "[error] %s (%.1fs, $%.4f)\n", s.Result, duration, s.CostUSD)
	} else {
		fmt.Fprintf(r.w, "[done] %.1fs, $%.4f\n", duration, s.CostUSD)
	}
}
