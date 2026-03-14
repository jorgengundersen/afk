# Logging

afk writes structured logs to a file each session. Logs are machine-readable, one event per line, with no log levels.

## Log File Location

Log files are written to `~/.local/share/afk/logs/` by default. Override with `--log`:

```
afk --log /tmp/afk-logs --beads
```

Each session creates a new file named `afk-YYYYMMDD-HHMMSS.log` using the UTC timestamp at startup.

## Structured Format

Every log line follows the same structure:

```
TIMESTAMP [afk] event-name key1=value1 key2="value with spaces"
```

- **Timestamp** — RFC 3339 UTC, always first
- **Tag** — `[afk]`, always second
- **Event name** — kebab-case, third position
- **key=value pairs** — remaining fields; values containing spaces are double-quoted

There are no DEBUG/INFO/WARN/ERROR levels. Every event is logged unconditionally. Verbosity is controlled by which events are emitted, not by filtering levels.

## Event Types

| Event | Fields | When |
|-------|--------|------|
| `session-start` | `mode`, `max` (if max-iter), `harness`, `beads`, `labels` (if set) | Session begins |
| `iteration-start` | `iteration`, `issue` (if beads), `title` (if beads) | Each loop iteration begins |
| `iteration-end` | `iteration`, `exit-code`, `duration` | Each loop iteration ends |
| `beads-check` | `count` | After querying `bd ready` |
| `sleeping` | `duration` | Daemon mode: entering sleep |
| `waking` | | Daemon mode: waking from sleep |
| `signal-received` | `signal` | SIGINT/SIGTERM caught |
| `session-end` | `reason`, `total-iterations`, `duration` | Session ends |

## Example: Max-Iterations Mode

A typical session with `afk --beads -n 3`:

```
2026-03-13T14:30:00Z [afk] session-start mode=max-iterations max=3 harness=claude beads=true
2026-03-13T14:30:00Z [afk] beads-check count=2
2026-03-13T14:30:00Z [afk] iteration-start iteration=1 issue=afk-42 title="Fix auth timeout"
2026-03-13T14:32:15Z [afk] iteration-end iteration=1 exit-code=0 duration=135s
2026-03-13T14:32:15Z [afk] beads-check count=1
2026-03-13T14:32:15Z [afk] iteration-start iteration=2 issue=afk-43 title="Add retry logic"
2026-03-13T14:34:00Z [afk] iteration-end iteration=2 exit-code=0 duration=105s
2026-03-13T14:34:00Z [afk] beads-check count=0
2026-03-13T14:34:00Z [afk] session-end reason=no-work total-iterations=2 duration=240s
```

afk exited early — no work remaining after iteration 2, so it stopped before reaching the max of 3.

## Example: Daemon Mode

A session with `afk -d --beads --sleep 60s`:

```
2026-03-13T14:30:00Z [afk] session-start mode=daemon harness=claude beads=true
2026-03-13T14:30:00Z [afk] beads-check count=1
2026-03-13T14:30:00Z [afk] iteration-start iteration=1 issue=afk-50 title="Update config parser"
2026-03-13T14:33:00Z [afk] iteration-end iteration=1 exit-code=0 duration=180s
2026-03-13T14:33:00Z [afk] beads-check count=0
2026-03-13T14:33:00Z [afk] sleeping duration=60s
2026-03-13T14:34:00Z [afk] waking
2026-03-13T14:34:00Z [afk] beads-check count=0
2026-03-13T14:34:00Z [afk] sleeping duration=60s
2026-03-13T14:35:00Z [afk] waking
2026-03-13T14:35:00Z [afk] beads-check count=1
2026-03-13T14:35:00Z [afk] iteration-start iteration=2 issue=afk-51 title="Fix CSV export"
2026-03-13T14:37:30Z [afk] iteration-end iteration=2 exit-code=0 duration=150s
2026-03-13T14:37:30Z [afk] signal-received signal=SIGINT
2026-03-13T14:37:30Z [afk] session-end reason=signal total-iterations=2 duration=450s
```

The daemon slept twice while idle, then worked on a new issue when one appeared. It shut down gracefully after receiving SIGINT.

## Terminal Output

Terminal output is separate from the log file. It is human-readable, not structured:

```
afk: starting (max-iterations=20, harness=claude, beads=on)
afk: [1/20] working on afk-42 "Fix auth bug"
afk: done (2 iterations, no work remaining)
```

### Output Modes

**Default** — status messages to stdout:

```
afk: starting (max-iterations=5, harness=claude, beads=on)
afk: [1/5] working on afk-42 "Fix auth timeout"
afk: done (1 iteration, 120s)
```

**`-v` (verbose)** — includes full agent output:

```
afk --beads -v -n 1
```

Shows the agent's stdout/stderr interleaved with afk's status messages. Useful for debugging prompt assembly or agent behavior.

**`--quiet`** — suppresses all output except errors:

```
afk --beads -d --quiet
```

No status messages. Only errors go to stderr. Suitable for background operation via cron or systemd.

**`--stderr`** — mirrors the structured log to stderr:

```
afk --beads --stderr
```

The same structured log lines written to the file are also printed to stderr. Useful for watching progress live with `tee` or a log viewer.

`--quiet` cannot be combined with `-v` or `--stderr`.
