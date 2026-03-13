# Error Handling

## Strategy

One strategy, applied consistently everywhere. Errors are values, not exceptions. No panics for expected failures.

## Error Creation

Errors are created at the point closest to the failure:

```go
// In beads/beads.go
if err := json.Unmarshal(data, &issues); err != nil {
    return nil, fmt.Errorf("parsing bd ready output: %w", err)
}
```

Use `fmt.Errorf` with `%w` to wrap. The message describes what this function was trying to do, not what went wrong (the wrapped error says that).

## Error Propagation

Most code propagates errors upward with added context:

```go
// In loop/maxiter.go
issues, err := b.Ready(ctx)
if err != nil {
    return fmt.Errorf("checking beads: %w", err)
}
```

Each layer adds one clause of context. The full chain reads as a sentence:
`checking beads: parsing bd ready output: invalid character 'x' in literal`

### Rules

- **Always wrap with context.** Never `return err` bare — the caller loses the stack of what was happening.
- **Never wrap twice at the same level.** One `fmt.Errorf` per return site.
- **Never log and return.** Do one or the other. Logging and returning causes duplicate noise. The handler (boundary) logs.

## Error Handling Boundaries

Errors are **handled** (logged, converted to exit code, presented to user) at exactly these boundaries:

1. **`cmd/afk/main.go`** — top-level. Catches errors from `cli/` and `loop/`, logs them, sets exit code.
2. **`loop/` orchestrator** — catches per-iteration errors from harness/beads. Logs them. Decides whether to continue or stop.

Everything else propagates.

```
harness.Run() → error
    ↓ propagate
loop.runIteration() → error
    ↓ handle: log, decide continue/stop
loop.Run() → error
    ↓ propagate
main() → handle: log, exit code
```

## Sentinel Errors

Define sentinel errors for conditions that callers need to distinguish:

```go
// In beads/beads.go
var (
    ErrNoWork     = errors.New("no work available")
    ErrBdNotFound = errors.New("bd not found in PATH")
)
```

Callers check with `errors.Is`:

```go
if errors.Is(err, beads.ErrNoWork) {
    // clean exit, not an error
}
```

Use sentinels sparingly. Only when the caller's behavior changes based on the error type.

## Structured Error Types

For errors that carry diagnostic data beyond a message:

```go
type ValidationError struct {
    Flag    string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("error: %s", e.Message)
}
```

Check with `errors.As`:

```go
var ve *cli.ValidationError
if errors.As(err, &ve) {
    fmt.Fprintln(os.Stderr, ve.Error())
    os.Exit(cli.ExitStartupError)
}
```

## Panics

Panics are reserved for programmer errors — states that should be impossible if the code is correct:

```go
switch mode {
case MaxIterations:
    // ...
case Daemon:
    // ...
default:
    panic(fmt.Sprintf("unreachable: unknown mode %d", mode))
}
```

Never `recover()` from panics. Let them crash. They indicate bugs that need fixing.

## Error Messages

Follow the product spec's error messages exactly. They are part of the contract:

```
error: --daemon and --max-iterations are mutually exclusive
error: --raw cannot be combined with --harness, --model, or --agent-flags
error: no prompt provided and beads integration is not active; nothing to do
error: harness "codex" not found in PATH; is it installed?
```

Format: `error: lowercase message with specific details`

No stack traces in user-facing output. No Go-style `harness: codex: exec: not found`. Human-readable sentences.

## Exit Codes

| Code | Constant | Meaning |
|------|----------|---------|
| 0 | `cli.ExitClean` | Normal completion |
| 1 | `cli.ExitStartupError` | Invalid flags, missing binary, bad config |
| 2 | `cli.ExitAllFailed` | Every iteration's agent returned non-zero |

Exit code is determined in `cmd/afk/main.go` based on the error (or lack thereof) returned from `loop.Run()`.
