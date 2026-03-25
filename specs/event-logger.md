# Event Logger

Structured event logging to a file. The loop and main record what happened;
the logger writes it. No decisions, no formatting opinions beyond the defined
format.

## What this adds

```
2026-03-25T14:00:00Z [afk] session-start mode=max-iter max=20 harness=claude beads=false
2026-03-25T14:00:01Z [afk] iteration-start iteration=1
2026-03-25T14:00:45Z [afk] iteration-end iteration=1 exit-code=0 duration=44s
2026-03-25T14:00:45Z [afk] session-end reason=complete total-iterations=1 duration=45s
```

Append structured log lines to a file. That's it.

## Structure

```
internal/logger/logger.go       # Logger type + Event func
internal/logger/logger_test.go  # Tests
```

## Log file location

`~/.local/share/afk/logs/afk-<timestamp>.log`

- Timestamp format in filename: `20060102-150405` (Go reference time).
- One file per session. The logger creates the file on first write.
- The logger creates the directory if it doesn't exist.

## Event format

```
<RFC3339 timestamp> [afk] <event-name> key=value key=value ...
```

- One line per event, newline terminated.
- Keys are alphabetically ordered for deterministic output.
- String values containing spaces are quoted.
- No nested structures. Flat key-value pairs only.

## Domain contract

```go
// Logger writes structured events to a log file.
type Logger struct { ... }

// New creates a logger that writes to the given path.
// The file is created on first Log call (lazy open).
func New(path string) *Logger

// Log writes a single event with the given name and fields.
func (l *Logger) Log(event string, fields map[string]string)

// Close flushes and closes the underlying file.
func (l *Logger) Close() error
```

- `Log` is the only write path. Callers build a `map[string]string` with
  the fields they want to log.
- `Log` must not return an error. Logging failures are silent — a broken log
  file must never crash the loop or affect the user's work. If the file can't
  be written, `Log` is a no-op.
- `Close` flushes any buffered writes and closes the file handle.

## Convenience: session path

```go
// SessionPath returns the log file path for a new session.
// Creates the log directory if needed.
func SessionPath() (string, error)
```

Called from main to determine where this session's log goes. Separated from
`New` so main can print the path or pass it around before the logger exists.

## Events

These are defined by the overview spec. The logger does not know about them —
it writes whatever event name and fields it receives. This table is for
reference:

| Event              | Fields                                          |
|--------------------|-------------------------------------------------|
| `session-start`    | mode, max (if max-iter), harness, beads          |
| `iteration-start`  | iteration, issue (if beads), title (if beads)   |
| `iteration-end`    | iteration, exit-code, duration, issue (if beads), title (if beads) |
| `beads-check`      | count                                           |
| `sleeping`         | duration                                        |
| `waking`           | -                                                |
| `error`            | message                                         |
| `session-end`      | reason, total-iterations, duration              |

The logger is generic. It does not validate event names or required fields.
That discipline belongs to the callers.

## Test cases

| Test                       | Input                                      | Expected                                          |
|----------------------------|--------------------------------------------|---------------------------------------------------|
| single event               | Log("test", {"k": "v"})                   | line: `<ts> [afk] test k=v\n`                     |
| multiple fields sorted     | Log("e", {"b": "2", "a": "1"})            | `<ts> [afk] e a=1 b=2\n`                          |
| value with spaces quoted   | Log("e", {"msg": "hello world"})           | `<ts> [afk] e msg="hello world"\n`                |
| no fields                  | Log("ping", {})                            | `<ts> [afk] ping\n`                               |
| multiple events appended   | Log twice                                  | two lines in file                                 |
| close flushes              | Log + Close, then read file                | event is present in file                          |
| session path creates dir   | SessionPath with missing dir               | dir created, path returned                        |

## Out of scope

- Log rotation or cleanup of old log files.
- Stderr mirroring or terminal output (the loop may print separately).
- Structured formats (JSON lines) — the current format is human-readable and
  grep-friendly. Revisit if machine parsing becomes a need.
- Event validation or required-field checking.

## Definition of done

- `Logger` writes structured events to a file in the specified format.
- `SessionPath` returns a timestamped path and creates the log directory.
- Log failures are silent (no panics, no returned errors from `Log`).
- All test cases pass.
- `go test ./internal/logger/...` passes.
- No global state — logger is instantiated and passed explicitly.
