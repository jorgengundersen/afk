# Agent Harness

Wraps external agent CLIs behind a common contract. Domain code calls one
function with a prompt and gets back an exit code. Everything CLI-specific
lives inside the harness wrapper — including how agent output is presented
to the user.

## What this adds

```
$ afk -p "do the thing"
# invokes: claude -p "do the thing"
# streams agent activity to terminal
# returns exit code from the subprocess

$ afk -p "do the thing" --harness-args "--dangerously-skip-permissions"
# invokes: claude -p "do the thing" --dangerously-skip-permissions
# returns exit code

$ afk -p "do the thing" --harness opencode
# invokes: opencode -p "do the thing"
# returns exit code

$ afk -p "do the thing" --raw "my-agent {prompt}"
# invokes: my-agent "do the thing"
# returns exit code
```

One function call per harness invocation. The loop does not know or care which
CLI is running underneath.

## Structure

```
internal/harness/           # Runner contract, implementations, output rendering
```

## Contract

Package `harness` exposes a Runner contract: given a context and a prompt
string, run the agent subprocess and return its exit code. An error means
failure to launch (binary not found, permission denied). A non-zero exit
code is NOT an error — it's a normal return value the loop uses to decide
what to do.

A factory function returns the correct Runner for the given configuration
(harness name, model, raw template, harness args). Unknown harness names
produce an error.

A pre-flight check verifies the harness binary exists in PATH before the
loop starts. This is a runtime/environment check, not a config constraint.

Each harness is responsible for showing the user what the agent is doing.
Agent activity is rendered to the terminal during execution. What "rendered"
means depends on the harness:

- When a harness supports structured output from the agent CLI, it parses
  the stream and renders each event as it arrives. The categories of
  information rendered are: agent text (formatted for terminal readability),
  tool invocations (tool name and key inputs, truncated), tool results
  (truncated), and an iteration summary (duration and cost when available).

- When structured output is not available, the harness passes subprocess
  stdout/stderr directly to the terminal.

Claude's CLI supports structured JSON streaming. The Claude harness uses
this to parse and render agent activity. OpenCode and Raw harnesses inherit
stdout/stderr directly (OpenCode structured output is future work).

Subprocesses run in their own process group so that context cancellation
terminates the entire tree, not just the direct child.

### Behaviour rules

| Given | Then |
|-------|------|
| harness="claude", model="" | Claude runner, no error |
| harness="claude", model="sonnet" | Claude runner passes `--model sonnet` to subprocess |
| harness="claude", harnessArgs="--dangerously-skip-permissions" | Args appended to subprocess command |
| harness="opencode" | OpenCode runner, no error |
| raw="my-agent {prompt}", prompt="hi" | Prompt is shell-escaped and substituted into template, executed via `sh -c` |
| raw is set | harnessArgs is ignored |
| Unknown harness name | Error: `unknown harness "<name>"` |
| Binary not in PATH | Error: `harness "<name>": binary "<bin>" not found in PATH` |
| Subprocess exits 0 | Return exitCode=0, err=nil |
| Subprocess exits non-zero | Return the exit code, err=nil |
| Binary cannot be started | Return err (not an exit code) |
| Context cancelled while running | Subprocess is terminated, return context error |
| Any subprocess is started | It runs in its own process group (PGID = child PID) |
| Context cancelled while subprocess is running | SIGTERM is sent to the entire process group, not just the direct child |
| Process group does not exit after SIGTERM within a grace period | SIGKILL is sent to the entire process group |
| Subprocess exits normally (no cancellation) | No signal is sent; exit code is returned |
| Subprocess has grandchildren when cancelled | All descendants in the process group are terminated |
| Running on a non-POSIX platform | Falls back to default os/exec behaviour |
| Claude harness runs subprocess | Structured JSON stream is parsed and rendered per the generic output model |
| Claude harness runs subprocess | Stderr is inherited by the terminal |
| OpenCode harness runs subprocess | Subprocess stdout/stderr are inherited by the terminal |
| Raw harness runs subprocess | Subprocess stdout/stderr are inherited by the terminal |
| Agent text contains raw content (newlines, whitespace) | Text is formatted for human-readable terminal output |
| Structured stream contains an unknown event | Event is skipped, warning written to stderr |
| Structured stream contains malformed JSON | Line is skipped, warning written to stderr |
| Context is cancelled mid-stream | Output processing stops promptly, partial output is fine |

### What this does NOT do

- Decide what flags the user passes via `--harness-args` (user's responsibility).
- Windows process group management — POSIX-only for now.
- Configurable grace period — use a sensible default.
- Timeout / stuck detection independent of context cancellation (future work).
- Parallel agent execution.
- Buffer or aggregate output events — each event is rendered as it arrives.
- Write agent output to the log file — logging is the logger's job.
- Configurable truncation limits for output rendering.

## Out of scope

- Timeout / stuck detection for agent subprocesses.
- Parallel agent execution.
- Validation of `--harness-args` content.
- Windows process group management.
- Color/formatting configuration or TTY detection for output.
- OpenCode structured output parsing (format not yet defined).

## Definition of done

- Runner contract: given a context and prompt, run agent, return exit code.
- Claude, OpenCode, and Raw implementations.
- Factory returns correct runner or error for unknown harness.
- Pre-flight binary check verifies binary exists in PATH.
- Subprocesses run in their own process group; cancellation kills the entire group.
- Claude harness renders structured agent activity to the terminal.
- OpenCode and Raw harnesses pass stdout/stderr through to the terminal.
- `go test ./internal/harness/...` passes.
