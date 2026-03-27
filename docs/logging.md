# Logging

afk writes a structured log file for each session so you can understand what
happened while you were away.

## Log File Location

```
~/.local/share/afk/logs/afk-YYYYMMDD-HHMMSS.log
```

The directory follows `$XDG_DATA_HOME/afk/logs/` if set, otherwise defaults
to `~/.local/share/afk/logs/`. A new file is created per session.

## Line Format

```
<RFC3339 timestamp> [afk] <event> [key=value ...]
```

Fields are sorted alphabetically. String values containing spaces are quoted.

Example:
```
2024-01-15T10:30:45Z [afk] session-start maxIter=20 mode=max-iterations
2024-01-15T10:30:46Z [afk] iteration-start issueID=bd-42 issueTitle="Fix auth bug" iteration=1
2024-01-15T10:31:12Z [afk] iteration-end duration=26.1s exitCode=0 issueID=bd-42 issueTitle="Fix auth bug" iteration=1
2024-01-15T10:31:12Z [afk] session-end reason=complete
```

## Event Types

| Event | Fields | Description |
|-------|--------|-------------|
| `session-start` | `mode`, `maxIter` | Session begins. Mode is `daemon` or `max-iterations` |
| `session-end` | `reason` | Session ends. Reason: `complete`, `signal`, or `no-work` |
| `iteration-start` | `iteration`, `issueID`?, `issueTitle`? | An iteration begins. Issue fields present when using beads |
| `iteration-end` | `iteration`, `exitCode`, `duration`, `issueID`?, `issueTitle`? | An iteration completes |
| `beads-check` | `hasWork` | Work source polled. `hasWork` is true/false |
| `work-source-error` | `err` | Work source failed (e.g., `bd ready` error) |
| `sleeping` | `duration` | Daemon entering sleep between cycles |
| `waking` | | Daemon waking from sleep |
| `error` | `iteration`, `err` | Runner failed to execute (binary not found, etc.) |

Fields marked with `?` are optional and only present when relevant.
