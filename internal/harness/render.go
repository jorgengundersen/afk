package harness

import (
	"context"
	"fmt"
	"io"
	"os"
)

const maxInputDisplay = 200

// Renderer writes parsed Claude events to a terminal-friendly text stream.
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

// Render writes a single event to the output.
func (r *Renderer) Render(ev Event) {
	switch ev.Type {
	case EventAssistant:
		r.renderAssistant(ev.Message)
	case EventToolResult:
		r.renderToolResult(ev.ToolResult)
	case EventResult:
		r.renderResult(ev.Result)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (r *Renderer) renderToolUse(block ContentBlock) {
	input := truncate(string(block.Input), maxInputDisplay)
	fmt.Fprintf(r.w, "[tool] %s: %s\n", block.Name, input)
}

func (r *Renderer) renderResult(rs *ResultSummary) {
	if rs == nil {
		return
	}
	duration := float64(rs.DurationMS) / 1000.0
	if rs.IsError {
		fmt.Fprintf(r.w, "[error] %s (%.1fs, $%.4f)\n", rs.Result, duration, rs.CostUSD)
	} else {
		fmt.Fprintf(r.w, "[done] %.1fs, $%.4f\n", duration, rs.CostUSD)
	}
}

func (r *Renderer) renderToolResult(tr *ToolResult) {
	if tr == nil {
		return
	}
	content := truncate(tr.Content, maxInputDisplay)
	fmt.Fprintf(r.w, "[result] %s\n", content)
}

func (r *Renderer) renderAssistant(msg *AssistantMessage) {
	if msg == nil {
		return
	}
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				fmt.Fprintln(r.w, block.Text)
			}
		case "tool_use":
			r.renderToolUse(block)
		}
	}
}
