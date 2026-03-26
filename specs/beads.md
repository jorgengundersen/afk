# Beads Client

Wraps the `bd` CLI behind a domain contract so the loop can fetch ready
issues without touching external types or subprocess details directly.

## What this adds

```
$ bd ready --json
# returns JSON array of issues sorted by priority

$ bd ready --json --label backend
# returns only issues with the "backend" label

$ bd ready --json --label-any frontend --label-any backend
# returns issues with "frontend" OR "backend" labels
```

The beads client executes these commands and returns the results as domain
types. afk's loop calls this client each iteration when `--beads` is active.

## Structure

```
internal/beads/
```

## Contract

Package `beads` provides a client that fetches ready issues from the `bd`
CLI. The client accepts optional label filters (AND and OR) and passes them
through as `--label` and `--label-any` flags to `bd ready --json`.

The client executes `bd ready` as a subprocess, parses the JSON output, and
returns a list of issues in domain types. Issues are returned in the order
`bd ready` provides (sorted by priority). An empty list means no work is
available.

The issue domain type contains the full set of fields returned by `bd ready`:
id, title, description, status, priority, issue type, owner, labels, and
dependencies. The raw JSON representation of the issue is also preserved so
callers can inject it into prompts without re-serializing.

A pre-flight check verifies the `bd` binary exists in PATH, similar to the
harness binary check. This runs before the loop starts.

### Behaviour rules

| Given | Then |
|-------|------|
| `bd ready` returns issues | List of domain-typed issues returned in priority order |
| `bd ready` returns empty array | Empty list returned (no error) |
| Label filters provided | Passed through as `--label` and `--label-any` flags to `bd ready` |
| Both AND and OR label filters provided | Both passed through to `bd ready` |
| No label filters provided | `bd ready --json` invoked without label flags |
| `bd` binary not in PATH | Pre-flight check returns error: `beads: binary "bd" not found in PATH` |
| `bd ready` subprocess exits non-zero | Error returned with context |
| `bd ready` returns malformed JSON | Error returned with context |
| Raw JSON of each issue preserved | Available to callers alongside parsed domain fields |

### What this does NOT do

- Claim or close issues — the agent does this via prompt instruction.
- Filter, sort, or re-rank issues — `bd ready` handles priority ordering.
- Validate issue content beyond JSON parsing.
- Cache results between calls.
- Manage the `bd` binary installation or configuration.

## Out of scope

- Writing back to beads (claiming, closing, updating).
- Issue filtering beyond labels (assignee, priority, type).
- Retry logic for `bd ready` failures — the loop handles retry at the
  iteration level.
- Custom `bd` binary path — expects `bd` in PATH.

## Definition of done

- Client fetches issues from `bd ready --json` and returns domain types.
- Label filters are passed through correctly.
- Pre-flight check verifies `bd` binary in PATH.
- Malformed JSON and subprocess failures return errors.
- Raw JSON preserved per issue.
- `go test ./internal/beads/...` passes.
