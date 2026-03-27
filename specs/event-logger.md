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
internal/logger/logger.go       # Logger type + Log func
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

## Contract

Package `logger` exposes a Logger that writes structured events to a log
file. The logger is created with a file path and lazily opens the file on
first write. It provides a Log method that accepts an event name and a map
of fields, formats them, and writes to the file.

Log returns an error. Callers decide how to handle write failures. This
follows design principle 4 (fail fast, fail loudly) — a broken logger must
be surfaced, not hidden.

Close flushes and closes the underlying file handle.

SessionPath returns the log file path for a new session and creates the log
directory if needed. Separated from logger creation so main can determine
the path before the logger exists.

### Behaviour rules

| Given | Then |
|-------|------|
| First Log call | File is lazily created at the configured path |
| Log directory does not exist | Directory is created recursively |
| Log is called with event and fields | Line written: `<RFC3339> [afk] <event> key=value ...` |
| Multiple fields provided | Keys are sorted alphabetically |
| Field value contains spaces | Value is quoted |
| No fields provided | Line contains only timestamp, prefix, and event name |
| Multiple Log calls | Lines are appended to the same file |
| File cannot be created (permissions, bad path) | Log returns an error |
| Write fails (disk full, I/O error) | Log returns an error |
| Log called after Close | No-op, no error |
| Close called twice | Second call returns nil |
| Close called before any Log | Returns nil (no file to close) |

### What this does NOT do

- Decide what to do about write failures — callers handle errors.
- Validate event names or required fields. That discipline belongs to callers.
- Format or transform field values beyond quoting spaces.
- Write to stderr or terminal.
- Rotate or clean up old log files.

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

## Out of scope

- Log rotation or cleanup of old log files.
- Stderr mirroring or terminal output (the loop may print separately).
- Structured formats (JSON lines) — the current format is human-readable and
  grep-friendly. Revisit if machine parsing becomes a need.
- Event validation or required-field checking.

## Definition of done

- Logger writes structured events to a file in the specified format.
- SessionPath returns a timestamped path and creates the log directory.
- Log returns an error on write failures — failures are never silent.
- Log after Close is a safe no-op.
- All test cases pass.
- `go test ./internal/logger/...` passes.
- No global state — logger is instantiated and passed explicitly.
