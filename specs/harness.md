# Agent Harness

Wraps external agent CLIs behind a common contract. Domain code calls one
function with a prompt and gets back an exit code. Everything CLI-specific
lives inside the harness wrapper.

## What this adds

```
$ afk -p "do the thing"
# invokes: claude -p "do the thing"
# returns exit code from the subprocess

$ afk -p "do the thing" --harness-args "--dangerously-skip-permissions --verbose"
# invokes: claude -p "do the thing" --dangerously-skip-permissions --verbose
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
internal/harness/harness.go       # Runner interface, registry, implementations
internal/harness/harness_test.go  # Tests (unit + integration where practical)
```

## Domain contract

```go
// Runner executes an agent with the given prompt and returns its exit code.
type Runner interface {
    Run(ctx context.Context, prompt string) (exitCode int, err error)
}
```

- `ctx` for cancellation (signal handling).
- `exitCode` is the subprocess exit code (0 = success).
- `err` is for failures to launch the process (binary not found, permission
  denied). A non-zero exit code from the agent is NOT an error — it's a normal
  return value the loop uses to decide what to do.

## Implementations

### Claude

Default harness. Invokes `claude` CLI in headless mode.

```
claude -p "<prompt>" [--model <model>] [<harness-args>...]
```

afk does not inject flags like `--dangerously-skip-permissions` by default.
The harness runs the agent with its own applied configuration. Users pass
additional flags explicitly via `--harness-args`.

If `Model` is set in config, add `--model <model>`.

### OpenCode

Invokes `opencode` CLI in headless mode.

```
opencode -p "<prompt>" [--model <model>] [<harness-args>...]
```

If `Model` is set, add `--model <model>`.

### Raw

Escape hatch. User provides a command template with `{prompt}` placeholder.

```
--raw "my-agent {prompt}"
```

`{prompt}` is replaced with the shell-escaped prompt. The resulting string is
executed via `sh -c`. Model flag is ignored (validated away by config).

## Registry

```go
// New returns a Runner for the given config.
func New(harness string, model string, raw string, harnessArgs string) (Runner, error)
```

- If `raw` is set, return a Raw runner (harness/model already validated away).
  `harnessArgs` is ignored for raw.
- Otherwise, look up `harness` by name. Return error for unknown names.
- `model` is passed through to the runner for harnesses that support it.
- `harnessArgs` is split and appended to the subprocess command as additional
  arguments. Passed through verbatim — no validation by afk.

Known harness names: `"claude"`, `"opencode"`.

Unknown name → `error: unknown harness "<name>"`.

## Validation (pre-flight)

The harness layer owns one validation check that config does not:

```go
// CheckBinary verifies the harness binary exists in PATH.
func CheckBinary(harness string, raw string) error
```

- For named harnesses: look up the binary name, check `exec.LookPath`.
- For raw: extract the first token from the template, check `exec.LookPath`.
- Returns: `harness "<name>": binary "<bin>" not found in PATH`.

This is called from main after config validation, before the loop starts.
It is NOT part of `config.Validate` — binary existence is a runtime/environment
check, not a config constraint.

## Test cases

### Unit tests

| Test                       | Input                              | Expected                              |
|----------------------------|------------------------------------|---------------------------------------|
| new claude runner          | harness="claude", model=""         | Claude runner, no error               |
| new claude with model      | harness="claude", model="sonnet"   | Claude runner with model set          |
| new claude with args       | harness="claude", harnessArgs="--dangerously-skip-permissions" | args appended to command |
| new opencode runner        | harness="opencode", model=""       | OpenCode runner, no error             |
| new raw runner             | raw="my-agent {prompt}"            | Raw runner, no error                  |
| unknown harness            | harness="nope"                     | error: unknown harness "nope"         |
| raw prompt substitution    | raw="cmd {prompt}", prompt="hi"    | command becomes `cmd "hi"`            |
| raw ignores harness args   | raw="cmd {prompt}", harnessArgs="--foo" | args ignored                    |

### Integration tests

Testing actual subprocess execution is inherently environment-dependent. Use
a test helper script or `echo` as a stand-in binary:

| Test                       | Setup                              | Expected                              |
|----------------------------|------------------------------------|---------------------------------------|
| run returns exit code 0    | harness wraps `true`               | exitCode=0, err=nil                   |
| run returns exit code 1    | harness wraps `false`              | exitCode=1, err=nil                   |
| binary not found           | harness wraps nonexistent binary   | exitCode=0, err=non-nil               |
| context cancellation       | cancel ctx during run              | err=non-nil (context cancelled)       |

## Process group management

Agents like `claude` or `opencode` may spawn child processes of their own. Without
process group isolation, context cancellation only kills the direct child —
grandchildren become orphan processes. The harness must prevent this.

### Behaviour rules

| Given | Then |
|-------|------|
| Any subprocess is started | It runs in its own process group (PGID = child PID) |
| Context is cancelled while subprocess is running | SIGTERM is sent to the entire process group, not just the direct child |
| Process group does not exit after SIGTERM within a grace period | SIGKILL is sent to the entire process group |
| Subprocess exits normally (no cancellation) | No signal is sent; exit code is returned as today |
| Subprocess has grandchildren when cancelled | All descendants in the process group are terminated |
| Running on a non-POSIX platform | Process group management is unavailable; falls back to default os/exec behaviour |

### What this does NOT do

- Windows process group management — POSIX-only for now.
- Configurable grace period — use a sensible default.
- Timeout / stuck detection independent of context cancellation (future work).

## Out of scope

- Timeout / stuck detection for agent subprocesses (future).
- Capturing agent stdout/stderr (the agent writes to the terminal directly).
- Parallel agent execution.
- Validation of `--harness-args` content (user's responsibility).
- Windows process group management (no `Setpgid` equivalent used yet).

## Definition of done

- `Runner` interface defined with `Run(ctx, prompt) (int, error)`.
- Claude, OpenCode, and Raw implementations.
- `New` factory returns correct runner or error for unknown harness.
- `CheckBinary` verifies binary exists in PATH.
- Subprocesses run in their own process group; cancellation kills the entire group.
- All test cases pass.
- `go test ./internal/harness/...` passes.
- Domain code (the loop, when it exists) depends only on `Runner`, never on
  subprocess types directly.
