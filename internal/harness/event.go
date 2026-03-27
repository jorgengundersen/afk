package harness

// EventKind discriminates common event types.
type EventKind string

const (
	KindText       EventKind = "text"
	KindToolCall   EventKind = "tool_call"
	KindToolOutput EventKind = "tool_output"
	KindSummary    EventKind = "summary"
)

// CommonEvent is the normalized event emitted by harness adapters
// and consumed by the shared renderer.
type CommonEvent struct {
	Kind       EventKind
	Text       *TextPayload
	ToolCall   *ToolCallPayload
	ToolOutput *ToolOutputPayload
	Summary    *SummaryPayload
}

// TextPayload carries agent thinking, reasoning, or response text.
type TextPayload struct {
	Content string
}

// ToolCallPayload carries the name and input summary of a tool invocation.
type ToolCallPayload struct {
	Name  string
	Input string
}

// ToolOutputPayload carries the result content from a tool invocation.
type ToolOutputPayload struct {
	Content string
}

// SummaryPayload carries iteration-complete metrics.
// Fields default to zero when the harness does not provide them.
type SummaryPayload struct {
	DurationMS   int
	CostUSD      float64
	InputTokens  int
	OutputTokens int
	Result       string
	IsError      bool
}
