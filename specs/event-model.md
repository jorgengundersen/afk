# Common Agent Event Model

A normalized event vocabulary that sits between harness-specific stream
adapters and the shared renderer. Each harness that supports structured output
provides an adapter that reads its wire format and emits common events. The
renderer consumes those events without knowing which harness produced them.

## What this adds

```
$ afk -p "refactor auth"
# Claude adapter parses stream-json → common events → shared renderer
[tool] Edit: src/auth.go...
[result] src/auth.go updated
[done] 12.3s, $0.0412

$ afk -p "refactor auth" --harness codex
# Codex adapter parses NDJSON → common events → shared renderer
[tool] shell: ls src/...
[result] auth.go main.go
[done] 8.1s, 1234 tokens in / 567 tokens out
```

Same renderer output regardless of which agent produced it. The user sees a
consistent view of agent activity across all first-class harnesses.

## Structure

```
internal/harness/           # Common event types, adapters, shared renderer
```

## Contract

The event model defines a small set of normalized event kinds that any
harness adapter can emit and the shared renderer can consume.

### Event kinds

| Kind         | Purpose                                  |
|--------------|------------------------------------------|
| `Text`       | Agent thinking, reasoning, or response   |
| `ToolCall`   | Agent is invoking a tool (name + input)  |
| `ToolOutput` | Tool returned a result                   |
| `Summary`    | Iteration complete (duration, cost, error, token usage) |

### Event structure

A single event type with a kind discriminator. Each kind has its own typed
payload (nil when not applicable):

- **Text** — contains the text content string.
- **ToolCall** — contains tool name and input summary string.
- **ToolOutput** — contains result content string.
- **Summary** — contains duration (ms), cost (USD), input/output token counts,
  result text, and error flag. Not all harnesses provide all fields — values
  default to zero when unavailable. Claude provides cost and duration. Codex
  provides token usage.

### Adapter contract

Each harness that supports structured output provides a parse function that:

- Reads newline-delimited JSON from a reader.
- Sends parsed common events to a channel.
- Closes the channel when the reader is exhausted or context is cancelled.
- Writes warnings for malformed or unknown lines (when a warning writer is
  provided).

### Renderer contract

The renderer accepts a channel of common events and writes formatted text to
a writer. It does not know or care which harness adapter produced the events.

### Harness mapping tables

**Claude → Common Events**

| Claude raw type                  | Common Event Kind | Notes                           |
|----------------------------------|-------------------|---------------------------------|
| `assistant` (text content block) | `Text`            |                                 |
| `assistant` (tool_use block)     | `ToolCall`        | name + JSON input               |
| `tool_result`                    | `ToolOutput`      | tool result content             |
| `result`                         | `Summary`         | cost_usd, duration_ms, is_error |

**Codex → Common Events**

| Codex raw event                              | Common Event Kind | Notes                                        |
|----------------------------------------------|-------------------|----------------------------------------------|
| `item.completed` → `agent_message`           | `Text`            | agent response text                          |
| `item.completed` → `reasoning`               | `Text`            | agent thinking/reasoning                     |
| `item.started` → `command_execution`         | `ToolCall`        | name="shell", input=command                  |
| `item.completed` → `command_execution`       | `ToolOutput`      | aggregated_output + exit_code                |
| `item.started` → `mcp_tool_call`             | `ToolCall`        | tool name + arguments                        |
| `item.completed` → `mcp_tool_call`           | `ToolOutput`      | tool result content                          |
| `item.completed` → `file_change`             | `ToolOutput`      | list of changed paths + kinds                |
| `item.started` → `web_search`                | `ToolCall`        | name="web_search", input=query               |
| `turn.completed`                             | `Summary`         | input_tokens, output_tokens                  |
| `turn.failed`                                | `Summary`         | IsError=true, Result=error message           |

Codex events not listed above (`thread.started`, `turn.started`, `todo_list`,
`collab_tool_call`, item-level `error`) are either informational or not
relevant to terminal rendering. Adapters may skip them or map them to `Text`
if useful context would otherwise be lost.

### Behaviour rules

| Given | Then |
|-------|------|
| Claude adapter receives an `assistant` event with text blocks | Emits one `Text` event per text block |
| Claude adapter receives an `assistant` event with tool_use blocks | Emits one `ToolCall` event per tool_use block |
| Claude adapter receives a `tool_result` event | Emits one `ToolOutput` event |
| Claude adapter receives a `result` event | Emits one `Summary` event with cost and duration |
| Codex adapter receives `item.completed` with `agent_message` | Emits one `Text` event |
| Codex adapter receives `item.completed` with `reasoning` | Emits one `Text` event |
| Codex adapter receives `item.started` with `command_execution` | Emits one `ToolCall` event |
| Codex adapter receives `item.completed` with `command_execution` | Emits one `ToolOutput` event |
| Codex adapter receives `turn.completed` | Emits one `Summary` event with token counts |
| Codex adapter receives `turn.failed` | Emits one `Summary` event with error flag set |
| Adapter receives malformed JSON line | Line is skipped, warning written if writer provided |
| Adapter receives unknown event type | Event is skipped, warning written if writer provided |
| Context is cancelled | Adapter stops promptly, channel is closed |
| Reader is exhausted | Channel is closed |
| Renderer receives `Text` event | Writes text content to output |
| Renderer receives `ToolCall` event | Writes tool name and truncated input |
| Renderer receives `ToolOutput` event | Writes truncated result content |
| Renderer receives `Summary` event | Writes available summary fields (duration, cost, tokens, error) |

### What this does NOT do

- Define harness-specific parsing logic (belongs in each adapter).
- Define rendering format or terminal styling (belongs in the renderer).
- Aggregate or buffer events — streaming, one at a time.
- Define the wire format of any external agent CLI.
- Write agent output to the log file (logging is a separate concern).

## Out of scope

- Adapter implementations for OpenCode or Copilot (future work — the adapter
  pattern supports them).
- Filtering or transforming events between adapter and renderer.
- Configurable truncation limits for rendered output.
- Structured logging of agent events (the event logger is a separate concern).

## Definition of done

- Common event type and payloads defined.
- Claude adapter emits common events (refactored from current implementation).
- Codex adapter emits common events.
- Shared renderer consumes common events.
- Existing Claude rendering tests pass with no behaviour change.
- Codex adapter has its own tests.
