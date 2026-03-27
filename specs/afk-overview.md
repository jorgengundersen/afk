# afk — Product Spec

## What

`afk` is a CLI that runs agentic coding loops unattended. You start it, walk
away, and come back to completed work.

It wraps agent CLIs (Claude Code, Codex, OpenCode, etc.) in a retry loop,
optionally pulling work from `bd` (beads) for self-directing operation.

## Modes

Two mutually exclusive modes per invocation:

- **Max-iterations** (default): Run up to N iterations (default 20), exit when
  done or no work remains.
- **Daemon** (`-d`): Run indefinitely. Sleep when idle, wake when work appears.

## Prompt Sources

A prompt must come from somewhere or there's nothing to do:

1. `-p` flag (highest priority)
1. Stdin pipe (when not a TTY)
1. Beads issue from `bd ready --json` (when `--beads` is active)

If none of the above → exit code 2.

When beads is active and an issue is found, the issue context is prepended to
the prompt with an instruction to claim, complete, and close the issue.

### Prompt Assembly

Prompt assembly is a pure function. Given a user prompt (string, possibly
empty), an optional issue, and an instruction string, it produces the final
prompt sent to the harness:

- Issue only: issue context + instruction
- Prompt only: user prompt verbatim
- Both: issue context + instruction + user prompt
- Neither: error (should never reach assembly — caught by validation)

## Agent Harnesses

A harness wraps an external CLI that afk invokes each iteration. Harnesses are
an external boundary — domain code interacts with them through a common
contract: given a prompt string, run the agent and return an exit code.

| Harness     | Binary     | Notes                                                        |
|-------------|------------|--------------------------------------------------------------|
| Claude Code | `claude`   | Default. Structured output via `--output-format stream-json` |
| Codex       | `codex`    | `codex exec` with `--json` for structured NDJSON output      |
| OpenCode    | `opencode` | Headless mode, stdout/stderr passthrough                     |
| Raw         | any        | Escape hatch: `--raw "cmd {prompt}"` — `{prompt}` is substituted |

**afk's responsibility per harness is exactly two things:** invoke the agent in
non-interactive mode, and enable structured output (when supported). All other
harness behaviour (sandbox, approvals, permissions) is the user's
responsibility via `--harness-args` or the agent's own config files.

**Working directory:** afk should be run in the directory it should operate on,
typically the project root. Harnesses inherit the current working directory.
afk does not support changing directories for harnesses.

Additional flags can be passed through to the harness CLI via `--harness-args`
(ignored for `--raw`).

`--raw` is mutually exclusive with `--harness` and `--model`.

## Subcommands

| Subcommand   | Description                                      |
|--------------|--------------------------------------------------|
| `quickstart` | Print an examples-only cheatsheet to stdout and exit |

Subcommands are intercepted before flag parsing. If the first argument matches
a subcommand, it runs and exits — no flags are processed.

## CLI Flags

| Flag        | Type     | Default | Description                             |
|-------------|----------|---------|-----------------------------------------|
| `-p`        | string   | —       | Prompt text                             |
| `-n`        | int      | 20      | Max iterations                          |
| `-d`        | bool     | false   | Daemon mode                             |
| `--sleep`   | duration | 60s     | Sleep between cycles (daemon only)      |
| `--harness` | string   | claude  | Which agent to use (claude, codex, opencode) |
| `--model`   | string   | —       | Model override mapped to harness's model flag |
| `--raw`     | string   | —       | Raw command with `{prompt}` placeholder |
| `--harness-args` | string | —  | Additional flags passed through to harness CLI |
| `--beads`   | bool     | false   | Pull work from bd                       |

## Behavior

### Iteration Cycle

Each iteration:

1. If beads active: fetch ready issues. No issues → stop (max-iter) or sleep
   (daemon).
1. Assemble prompt (pure — see Prompt Assembly above).
1. Invoke harness.
1. Log result (exit code, duration, issue ID/title if beads).
1. Agent failure (non-zero exit) does NOT stop the loop — log and continue.

### Signal Handling

- SIGINT/SIGTERM: cancel context, let current agent finish, exit 0.

### Exit Codes

| Code | Meaning                                |
|------|----------------------------------------|
| 0    | Clean exit                             |
| 1    | Runtime error or all iterations failed |
| 2    | CLI usage error                        |

## Logging

Always logs to `~/.local/share/afk/logs/afk-<timestamp>.log`.

Format: `<timestamp> [afk] <event> key=value ...`

Events:

| Event              | Fields                                          |
|--------------------|-------------------------------------------------|
| `session-start`    | mode, max (if max-iter), harness, beads          |
| `iteration-start`  | iteration, issue (if beads), title (if beads)   |
| `iteration-end`    | iteration, exit-code, duration, issue (if beads), title (if beads) |
| `beads-check`      | count                                           |
| `sleeping`         | duration                                        |
| `waking`           | —                                                |
| `error`            | message                                         |
| `session-end`      | reason, total-iterations, duration              |

## Validation

All validation is fail-fast: checked before the loop starts, exits with
code 2 and an actionable error message.

| Condition                      | Error message                                                              |
|--------------------------------|----------------------------------------------------------------------------|
| `--raw` + `--harness`/`--model` | `--raw cannot be combined with --harness or --model`                       |
| `--sleep` without `-d`        | `--sleep requires daemon mode (-d)`                                        |
| No prompt and no beads        | `no prompt provided and beads not active; nothing to do`                   |
| Harness binary not in PATH    | `harness "<name>": binary "<bin>" not found in PATH`                       |
| Unknown harness name          | `unknown harness "<name>"`                                                 |

## External Boundaries

These are the system's external dependencies. Each is isolated behind its own
contract so domain code never touches external types directly.

| Boundary       | External system | Domain contract                                |
|----------------|-----------------|------------------------------------------------|
| Agent harness  | CLI subprocess  | Given prompt → run agent → return exit code    |
| Beads client   | `bd` CLI        | Fetch ready issues → return list of issues     |
| Event logger   | Filesystem      | Record structured events to log file           |

## Future: Config File

Not implemented yet, but the core must be designed so this can be added without
refactoring domain code.

- Global: `$HOME/.config/afk/config.toml`
- Project: `.afk.config.toml` in project root

**Precedence** (highest wins): CLI flags → project config → global config → defaults.

**Design constraint:** `Config` is a plain data struct. The loop takes a
`Config` value and does not care how it was populated. Flag parsing is just one
source. Adding config file loading later means adding another source + a merge
step before calling `Run()`. No changes to domain code.

## Non-Goals (for now)

- Parallel agents
- TUI dashboard
- Agent timeout / stuck detection
- Notifications
