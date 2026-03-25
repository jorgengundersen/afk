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
internal/harness/harness.go           # Runner contract, registry, implementations
internal/harness/harness_test.go      # Tests
internal/harness/runcmd_unix.go       # Process group management (Unix)
internal/harness/runcmd_other.go      # Fallback (non-Unix)
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

## Agent output

Each harness is responsible for showing the user what the agent is doing.
The generic contract is: agent activity is rendered to the terminal during
execution. What "rendered" means depends on the harness.

### Generic output model

All harnesses present the same categories of information when available:

| Category | What the user sees |
|----------|--------------------|
| Agent text | What the agent is saying or thinking, formatted for terminal readability |
| Tool invocation | Tool name and key inputs (truncated for readability) |
| Tool result | Output from the tool (truncated for readability) |
| Iteration summary | Duration and cost (if the agent reports it) |

Not all harnesses support all categories. When structured output is not
available, the harness falls back to passing subprocess stdout/stderr
directly to the terminal.

### Per-harness output

**Claude** — requests structured JSON output from the CLI
(`--output-format stream-json --verbose`). Parses the stream and renders
each event as it arrives according to the generic model above. Stderr is
inherited.

**OpenCode** — inherits stdout/stderr directly. No structured output
parsing (format not yet defined). Future work.

**Raw** — inherits stdout/stderr directly. No processing. The user's
command is responsible for its own output.

### Output behaviour rules

| Given | Then |
|-------|------|
| Harness supports structured output | Agent activity is parsed and rendered per the generic model |
| Agent text contains raw content (newlines, whitespace) | Text is formatted for human-readable terminal output |
| Harness does not support structured output | Subprocess stdout/stderr are inherited by the terminal |
| Structured stream contains an unknown event | Event is silently skipped |
| Structured stream contains malformed JSON | Line is skipped, processing continues |
| Context is cancelled mid-stream | Output processing stops promptly, partial output is fine |

## Process group management

Agents like `claude` or `opencode` may spawn child processes of their own.
Without process group isolation, context cancellation only kills the direct
child — grandchildren become orphan processes. The harness must prevent this.

| Given | Then |
|-------|------|
| Any subprocess is started | It runs in its own process group (PGID = child PID) |
| Context is cancelled while subprocess is running | SIGTERM is sent to the entire process group, not just the direct child |
| Process group does not exit after SIGTERM within a grace period | SIGKILL is sent to the entire process group |
| Subprocess exits normally (no cancellation) | No signal is sent; exit code is returned |
| Subprocess has grandchildren when cancelled | All descendants in the process group are terminated |
| Running on a non-POSIX platform | Falls back to default os/exec behaviour |

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
