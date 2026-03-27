# Wiring

Composes the internal packages into a running CLI. All domain primitives
(config, signal, logger, prompt, harness, loop) exist and are independently
tested. This module connects them in `main.go` so that `afk` actually runs
agent loops instead of printing the prompt and exiting.

## What this adds

```
$ afk -p "refactor auth" -n 3
# runs claude 3 times with "refactor auth"
# session logged to ~/.local/share/afk/logs/afk-<ts>.log
# exits 0

$ afk -p "do work" -d
# runs claude in daemon mode until Ctrl+C
# exits 0

$ afk -p "do work" --harness codex
# runs codex exec with structured NDJSON output

$ afk -p "do work" --harness opencode
# runs opencode instead of claude

$ afk --raw "my-agent {prompt}" -p "fix bug"
# runs my-agent via sh -c with prompt substituted
```

## Structure

```
cmd/afk/main.go   # Entry point: compose and run
```

No new packages. No new files beyond what exists. This is purely wiring
existing packages together in `main.go`.

## Contract

`main.go` is the composition root. It connects the existing packages — config,
signal, logger, prompt, harness, and loop — threading each package's output
into the next package's input. Any setup failure exits immediately with an
actionable error. When setup succeeds, the loop runs to completion and its
exit code becomes the process exit code.

The logger is always closed before exit, regardless of how the process ends.

### Behaviour rules

| Given | Then |
|-------|------|
| Flag parsing fails | Print error to stderr, exit 2 |
| Config validation fails | Print error to stderr, exit 2 |
| Session path creation fails | Print error to stderr, exit 1 |
| Prompt assembly fails | Print error to stderr, exit 2 |
| Harness binary not in PATH | Print error to stderr, exit 2 |
| Harness factory returns error | Print error to stderr, exit 2 |
| All setup succeeds | Run loop, exit with loop's exit code |
| Signal received during loop | Context cancelled, loop exits cleanly |
| Log fails during loop | Loop exits with code 1 — unobservable session is a failure |
| Loop completes | Logger is closed before exit |
| Logger close fails | Ignored (best effort) |

### What this does NOT do

- Introduce new types, interfaces, or packages.
- Add any domain logic — all logic lives in the existing packages.
- Handle stdin pipe as prompt source (future work).
- Handle beads issue fetching (future work — the loop will gain this).
- Print the log file path to the user (the logger is silent; future
  improvement if needed).
- Add terminal output beyond what the harness already provides.

## Out of scope

- Config file loading and merge (future — designed for, not implemented).
- Stdin pipe prompt source.
- Beads integration in the loop.
- Any new error handling or retry logic.
- Tests for main wiring beyond the existing `main_test.go` smoke tests.

## Definition of done

- `afk -p "hello" -n 1` runs the harness once and exits (not just prints).
- Signal handling is active — Ctrl+C during a run exits cleanly.
- A session log file is created with session-start and session-end events.
- `go build ./cmd/afk` succeeds.
- `go test ./...` passes.
