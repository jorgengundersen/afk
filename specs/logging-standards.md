# Logging Standards

## Format

Structured key=value pairs. One line per event. No JSON, no free-form text.

```
2026-03-13T14:30:00Z [afk] event-name key1=value1 key2="value with spaces"
```

### Fields

- **Timestamp**: RFC 3339 UTC. Always first.
- **Tag**: `[afk]`. Always second.
- **Event name**: kebab-case. Third position.
- **Key=value pairs**: remaining fields. Values with spaces are double-quoted.

## Events

These are the defined events. No ad-hoc logging. Every log line matches one of these.

| Event | Required Fields | When |
|-------|----------------|------|
| `session-start` | `mode`, `max` (if max-iter), `harness`, `beads`, `labels` (if set) | Session begins |
| `session-end` | `reason`, `total-iterations`, `duration` | Session ends |
| `iteration-start` | `iteration`, `issue` (if beads), `title` (if beads) | Each loop iteration begins |
| `iteration-end` | `iteration`, `exit-code`, `duration` | Each loop iteration ends |
| `beads-check` | `count` | After querying bd ready |
| `sleeping` | `duration` | Daemon mode: entering sleep |
| `waking` | | Daemon mode: waking from sleep |
| `signal-received` | `signal` | SIGINT/SIGTERM caught |
| `error` | `message` | Runtime error that doesn't stop the loop |

## Implementation

### Logger type (in `eventlog` package)

```go
type Logger struct {
    file   *os.File
    stderr bool // mirror to stderr
}

func (l *Logger) Event(name string, fields ...Field)
```

`Field` is a key-value pair. The logger formats and writes. No `Printf`-style formatting — callers pass structured data.

### No levels

No DEBUG/INFO/WARN/ERROR levels. Every event is logged. Verbosity is controlled by which events are emitted, not by filtering levels. The `--verbose` flag adds agent output capture, not more log levels.

### Thread safety

The logger is safe for concurrent use (single goroutine writes, or mutex-protected). In practice, `afk` is single-threaded in v1, but the logger should not break if that changes.

## Terminal Output

Separate from the log file. Terminal output is human-facing, not structured:

```
afk: starting (max-iterations=20, harness=claude, beads=on)
afk: [1/20] working on afk-42 "Fix auth bug"
afk: done (2 iterations, no work remaining)
```

Terminal output respects `--quiet` (suppress all except errors) and `--verbose` (show agent output).

Terminal output goes to **stdout**. Errors go to **stderr**. `--stderr` mirrors the structured log to stderr.

## Where to Instrument

Following the architectural principle: observe at edges, pure code is silent.

- **Log in `internal/loop/`**: session start/end, iteration start/end, beads checks, sleep/wake, signal received.
- **Log in `cmd/afk/main.go`**: startup errors.
- **Do NOT log in**: `internal/prompt/` (pure), `internal/config/` (pure), `internal/harness/` (called from loop which logs), `internal/beads/` (called from loop which logs).

If `internal/harness/` or `internal/beads/` encounter errors, they return them. `internal/loop/` decides whether and how to log.
