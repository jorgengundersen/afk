# User Guide

## Concepts

afk wraps an agent CLI in a retry loop. Each iteration assembles a prompt,
invokes the agent, and records structured logs. You type `afk`, walk away,
and come back to completed work.

## Two Modes

### Max-iterations (default)

Runs up to `-n` iterations (default 20) and exits. Iterations run
back-to-back with no delay.

```sh
afk -p "refactor the auth middleware"
afk -n 5 -p "fix the failing tests"
```

### Daemon

Runs indefinitely, sleeping between cycles. Use this for a self-directing
work loop that pulls from beads.

```sh
afk -d --beads --sleep 5m
```

The default sleep is 60 seconds. Daemon mode responds to signals immediately,
even mid-sleep.

## Prompt Sources

A prompt can come from multiple sources:

**Inline text:**
```sh
afk -p "add input validation to the API"
```

**File:**
```sh
afk -p instructions.md
```

**Stdin:**
```sh
echo "fix the failing tests" | afk
```

**Beads:** When `--beads` is enabled and no `-p` is given, the prompt comes
entirely from the next ready bead.

You can combine `-p` with `--beads` — the user prompt and issue context are
both included in the final prompt sent to the agent.

## Beads Integration

When `--beads` is enabled, afk queries `bd ready --json` at the start of
each iteration to find the highest-priority unblocked issue.

The issue's full JSON is included in the prompt along with an instruction
telling the agent to claim the issue before starting and close it when done.
The agent handles claiming and closing via the `bd` CLI — afk itself does
not call `bd update --claim` or `bd close`.

If no work is available:
- In max-iterations mode, afk exits cleanly (exit code 0).
- In daemon mode, afk sleeps and retries.

### Label Filtering

Filter which beads the agent works on:

```sh
# AND: all labels must match
afk -d --beads --label backend --label auth

# OR: at least one label must match
afk -d --beads --label-any frontend --label-any mobile

# Combined
afk -d --beads --label team-alpha --label-any bug --label-any feature
```

Both flags are repeatable — specify as many as needed.

## Practical Tips

**Start small.** Use `-n 1` for a single iteration to verify the agent
handles your prompt well before running a full loop.

```sh
afk -n 1 -p "update the changelog"
```

**Check logs.** Each session writes a structured log to
`~/.local/share/afk/logs/`. Use these to understand what happened while
you were away.

**Use a different agent.** Switch harnesses with `--harness`:

```sh
afk -p "add tests" --harness codex
```

**Pass harness-specific flags.** Use `--harness-args` for agent config
that afk doesn't manage:

```sh
afk -p "refactor" --harness-args "--dangerously-skip-permissions"
```

**Run in the right directory.** afk does not change the working directory —
agents inherit wherever you run afk from. Navigate to your project root first.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Clean exit or no work available |
| 1 | Runtime error or all iterations failed |
| 2 | CLI usage or validation error |
