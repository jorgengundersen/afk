# User Guide

`afk` runs agentic coding loops unattended. Type `afk`, walk away, come back to completed work.

## The Basic Loop

Each iteration follows the same steps:

1. **Check for work** — if beads integration is active, query `bd ready --json` for the next issue
2. **Assemble the prompt** — combine any issue context, instruction text, and your prompt into one final prompt
3. **Invoke the agent** — hand the prompt to the configured harness (e.g. Claude Code)
4. **Log the result** — write a structured log entry with iteration number, issue ID, exit code, and duration
5. **Repeat or exit** — continue to the next iteration, or stop if the exit condition is met

If the agent fails on a given iteration, afk logs the error and moves on to the next iteration — a single failure does not stop the loop.

## Max-Iterations Mode

This is the default mode. afk runs up to N iterations (default 20), then exits.

```
afk -p "refactor the auth module" -n 5
```

Early exit happens when beads is active and there are no more issues — afk prints "no work remaining" and exits cleanly (exit code 0) rather than burning through remaining iterations.

Use max-iterations mode when you have a bounded amount of work and want afk to stop on its own.

## Daemon Mode

Daemon mode runs indefinitely, sleeping when idle and waking when work appears.

```
afk -d --beads
```

When no work is available, afk sleeps for 60 seconds (configurable with `--sleep`), then checks again. When work is found, it invokes the agent immediately. After the agent finishes, it checks for more work right away — no sleep between consecutive tasks.

Use daemon mode for continuous integration of a work queue, where issues arrive over time.

```
afk -d --beads --sleep 5m     # check every 5 minutes when idle
```

Stop a daemon with Ctrl-C (see [Signal Handling](#signal-handling)).

## Prompt Assembly

The final prompt sent to the agent is built from up to three parts, in this order:

1. **Issue block** (if beads found an issue) — the issue ID, title, description, and acceptance criteria formatted as a structured block
2. **Instruction text** — a default instruction telling the agent to claim, complete, and close the issue (customizable with `--beads-instruction`)
3. **User prompt** (if provided) — your text from `--prompt` or stdin

### Example

Given an issue `afk-42 "Fix auth timeout"` and `--prompt "use exponential backoff"`, the agent receives:

```
## Issue afk-42: Fix auth timeout

<issue JSON with title, description, acceptance criteria>

Claim this issue and complete it. Follow AGENTS.md instructions. When complete, close the issue and exit.

use exponential backoff
```

You can use any combination: issue-only (beads without a prompt), prompt-only (no beads), or both.

## Beads Integration

[Beads](https://github.com/jorgengundersen/bd) (`bd`) is a lightweight issue tracker with dependency support. afk integrates with it to automatically pick up and work on issues.

Enable it with `--beads`:

```
afk --beads -p "work on the next issue"
```

How it works:

- afk runs `bd ready --json` each iteration to get the highest-priority unblocked issue
- The first issue returned is selected (bd returns them in priority order)
- The issue is injected into the prompt so the agent has full context
- After the agent finishes, the next iteration fetches the next ready issue

### Label Filtering

Filter to specific labels with `--beads-labels` (implies `--beads`):

```
afk --beads-labels "backend,p0"    # only issues with these labels
```

### Custom Instructions

Override the default instruction text appended with each issue:

```
afk --beads --beads-instruction "Fix this bug. Write tests. Do not refactor unrelated code."
```

### Prerequisites

`bd` must be installed and configured in the current repository. afk calls it as a subprocess.

## Signal Handling

afk handles shutdown signals gracefully:

- **First SIGINT (Ctrl-C) or SIGTERM**: afk waits for the current agent invocation to finish, then exits cleanly. No new iteration starts.
- **Second SIGINT**: force quit immediately.

This ensures the agent has a chance to finish its current task — committing partial work, closing issues, etc. — before afk shuts down.

## Tips

### Use AGENTS.md for project instructions

Place an `AGENTS.md` file in your repository root with project-specific instructions. Agents like Claude Code read this file automatically, so it steers their behavior within the afk loop — coding standards, testing requirements, commit conventions, etc.

### Pipe prompts from stdin

```
echo "fix all TODO comments" | afk -n 10
```

### Watch progress live

Use `--stderr` to mirror log output to stderr while the loop runs:

```
afk --beads --stderr
```

### Target a different repository

Use `-C` to run afk in a different directory without changing your shell's working directory:

```
afk -C ~/projects/other-repo --beads -d
```

### Combine with verbose mode

Use `-v` to see full agent output and issue JSON for debugging:

```
afk --beads -v -n 1
```

### Keep it quiet

Use `--quiet` to suppress all terminal output except errors:

```
afk --beads -d --quiet
```
