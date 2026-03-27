# Agent Harness

Wraps external agent CLIs behind a common contract. Domain code calls one
function with a prompt and gets back an exit code. Everything CLI-specific
lives inside the harness wrapper — including how agent output is presented
to the user.

## Principles

**afk's responsibility per harness is exactly two things:**

1. Invoke the agent in **non-interactive mode** (no TTY, no human prompts).
2. Enable **structured output** so afk can parse and render agent activity.

Everything else — sandbox policy, approval mode, permissions, model provider
configuration — is the user's responsibility. Users configure harness behaviour
via `--harness-args` or the agent's own config files. afk does not inject,
default, or override any harness-specific behavioural flags.

**afk's flag contract** (`--model`, `--harness-args`, etc.) is afk's own
interface. Each harness adapter maps afk's flags to the correct CLI flags for
its agent. If a harness uses `--llm` instead of `--model`, the adapter handles
the translation — the user always uses afk's flags.

**Working directory:** afk does not support changing the working directory for
harnesses. afk should be run in the directory it should operate on (typically
the project root). Harnesses inherit the current working directory. Any
`--cd`-style functionality provided by agent CLIs is not exposed or managed
by afk.

## What this adds

```
$ afk -p "do the thing"
# invokes: claude -p "do the thing" --output-format stream-json
# streams agent activity to terminal via common event model
# returns exit code from the subprocess

$ afk -p "do the thing" --harness-args "--dangerously-skip-permissions"
# invokes: claude -p "do the thing" --dangerously-skip-permissions --output-format stream-json
# returns exit code

$ afk -p "do the thing" --harness codex
# invokes: codex exec "do the thing" --json
# streams agent activity to terminal via common event model
# returns exit code

$ afk -p "do the thing" --harness codex --harness-args "--full-auto"
# invokes: codex exec "do the thing" --json --full-auto
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

- When a harness supports structured output, it parses the agent's stream
  through a harness-specific adapter that emits common events (see
  `specs/event-model.md`). A shared renderer consumes those events and writes
  formatted text to the terminal. The adapter pattern means adding a new
  harness with structured output requires only a new parser — the renderer
  is reused.

- When structured output is not available, the harness passes subprocess
  stdout/stderr directly to the terminal.

Subprocesses run in their own process group so that context cancellation
terminates the entire tree, not just the direct child.

## Harness implementations

### Claude (default)

- **Binary:** `claude`
- **Non-interactive flag:** `-p "<prompt>"` (headless prompt mode)
- **Structured output flag:** `--output-format stream-json`
- **Model passthrough:** `--model <model>` (when afk's `--model` is set)
- **Additional args:** `--harness-args` appended to command

### Codex

- **Binary:** `codex`
- **Subcommand:** `exec` (non-interactive execution mode)
- **Non-interactive:** `exec` subcommand is inherently non-interactive
- **Structured output flag:** `--json` (newline-delimited JSON events)
- **Prompt:** Positional argument to `exec` (not a `-p` flag) — the Codex
  adapter builds its own command, it does not share argument construction
  with other harnesses
- **Model passthrough:** `--model <model>` (when afk's `--model` is set)
- **Additional args:** `--harness-args` appended to command

### OpenCode

- **Binary:** `opencode`
- **Non-interactive flag:** `-p "<prompt>"` (headless mode)
- **Structured output:** Not available — stdout/stderr inherited by terminal
- **Model passthrough:** `--model <model>` (when afk's `--model` is set)
- **Additional args:** `--harness-args` appended to command

### Raw

- **Template:** User-provided command string with `{prompt}` placeholder
- **Execution:** `sh -c` with shell-escaped prompt substitution
- **Structured output:** Not available — stdout/stderr inherited by terminal
- **`--harness-args`:** Ignored (user controls the full command template)

## Behaviour rules

| Given | Then |
|-------|------|
| harness="claude", model="" | Claude runner, no error |
| harness="claude", model="sonnet" | Claude runner passes `--model sonnet` to subprocess |
| harness="claude", harnessArgs="--dangerously-skip-permissions" | Args appended to subprocess command |
| harness="codex", model="" | Codex runner, invokes `codex exec "<prompt>" --json` |
| harness="codex", model="o3" | Codex runner passes `--model o3` to subprocess |
| harness="codex", harnessArgs="--full-auto" | Args appended to subprocess command |
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
| Claude harness runs subprocess | Structured JSON stream parsed via Claude adapter, rendered via shared renderer |
| Codex harness runs subprocess | Structured NDJSON stream parsed via Codex adapter, rendered via shared renderer |
| Claude or Codex harness runs subprocess | Stderr is inherited by the terminal |
| OpenCode harness runs subprocess | Subprocess stdout/stderr are inherited by the terminal |
| Raw harness runs subprocess | Subprocess stdout/stderr are inherited by the terminal |
| Agent text contains raw content (newlines, whitespace) | Text is formatted for human-readable terminal output |
| Structured stream contains an unknown event | Event is skipped, warning written to stderr |
| Structured stream contains malformed JSON | Line is skipped, warning written to stderr |
| Context is cancelled mid-stream | Output processing stops promptly, partial output is fine |

### What this does NOT do

- Configure harness behaviour (sandbox, approvals, permissions) — user's
  responsibility via `--harness-args` or agent config files.
- Decide what flags the user passes via `--harness-args`.
- Manage or propagate working directory — harnesses inherit cwd.
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
- OpenCode structured output parsing (future work — the adapter pattern
  supports adding it when the format is defined).
- Copilot harness (future work — same adapter pattern applies).

## Definition of done

- Runner contract: given a context and prompt, run agent, return exit code.
- Claude, Codex, OpenCode, and Raw implementations.
- Factory returns correct runner or error for unknown harness.
- Pre-flight binary check verifies binary exists in PATH.
- Subprocesses run in their own process group; cancellation kills the entire group.
- Claude and Codex harnesses parse structured output via harness-specific
  adapters into common events, rendered via the shared renderer.
- OpenCode and Raw harnesses pass stdout/stderr through to the terminal.
- `go test ./internal/harness/...` passes.
