# `afk` — Product Specification

## 1. Overview

`afk` is a CLI tool that runs agentic coding loops unattended. The user starts `afk`, walks away, and returns to find completed work — with a log of everything that happened.

Inspired by the "Ralph Wiggum Loop" pattern by Geoffrey Huntley (`while :; do cat PROMPT.md | claude-code; done`), `afk` replaces that prototype with a robust, configurable tool supporting multiple agent harnesses, two runtime modes, structured logging, and soft integration with `bd` (beads) for issue tracking.

**Core value:** Type `afk`, leave, come back to completed work.

---

## 2. Runtime Modes

`afk` operates in exactly one of two modes per invocation:

- **Max-iterations mode** (default): Run the loop up to N times (default 20), then exit. Exits early if no work remains.
- **Daemon mode**: Run indefinitely. Sleep when no work is available; wake and resume when work appears. Default sleep interval: 60 seconds.

These are mutually exclusive.

---

## 3. Agent Harnesses

An agent harness is the external CLI tool `afk` invokes each iteration. Supported at launch:

| Harness | Binary | Notes |
|---|---|---|
| Claude Code | `claude` | Default |
| OpenCode | `opencode` | |
| Codex | `codex` | OpenAI Codex CLI |
| GitHub Copilot CLI | `copilot` | |

Additionally, a **raw command** escape hatch lets the user specify an arbitrary command string, bypassing harness abstraction entirely. The literal `{prompt}` in the raw command is replaced with the assembled prompt.

All harnesses run **headless by default**.

---

## 4. CLI Interface

### Synopsis

```
afk [flags]
```

No subcommands. Single-command tool.

### Flags

#### Mode

| Flag | Type | Default | Description |
|---|---|---|---|
| `-n`, `--max-iterations` | int | `20` | Max loop iterations |
| `-d`, `--daemon` | bool | `false` | Daemon mode (loop indefinitely) |
| `--sleep` | duration | `60s` | Sleep interval in daemon mode when no work available |

#### Agent Harness

| Flag | Type | Default | Description |
|---|---|---|---|
| `-H`, `--harness` | string | `claude` | Harness: `claude`, `opencode`, `codex`, `copilot` |
| `-m`, `--model` | string | _(harness default)_ | Model to pass to the harness |
| `--agent-flags` | string | _(none)_ | Extra flags passed verbatim to the harness CLI |
| `--raw` | string | _(none)_ | Raw command string (mutually exclusive with `--harness`, `--model`, `--agent-flags`) |

#### Prompt

| Flag | Type | Default | Description |
|---|---|---|---|
| `-p`, `--prompt` | string | _(none)_ | Prompt text for the agent |

- Supports shell substitution: `-p "$(cat PROMPT.md)"`
- Stdin is read when `-p` is absent and stdin is not a TTY: `cat PROMPT.md | afk`
- Precedence: `--prompt` > stdin > beads-only (if active)
- If no prompt and beads not active → error

#### Beads Integration

| Flag | Type | Default | Description |
|---|---|---|---|
| `--beads` | bool | `false` | Enable beads integration via `bd ready --json` |
| `--beads-labels` | string | _(none)_ | Comma-separated label filter. Implies `--beads` |
| `--beads-instruction` | string | _(see below)_ | Override the standard instruction text |

Default instruction text:
> Claim this issue and complete it. Follow AGENTS.md instructions. When complete, close the issue and exit.

#### Logging

| Flag | Type | Default | Description |
|---|---|---|---|
| `--log` | path | `~/.local/share/afk/logs/` | Directory for session log files |
| `--stderr` | bool | `false` | Mirror log output to stderr (in addition to log file) |
| `-v`, `--verbose` | bool | `false` | Increased verbosity (agent output, full issue JSON) |
| `-q`, `--quiet` | bool | `false` | Suppress all terminal output except errors |

### Examples

```bash
# Basic: 20 iterations with a prompt file
afk -p "$(cat PROMPT.md)"

# Daemon mode with beads label filter
afk --daemon --beads-labels "backend,p0"

# 5 iterations with Codex
afk -n 5 --harness codex --model gpt-4o -p "Fix all lint warnings"

# Raw command passthrough (e.g., aider)
afk --raw "aider --yes-always {prompt}" -p "Refactor the auth module"

# Pipe prompt, beads on, watch logs in terminal
cat PROMPT.md | afk --beads --stderr

# Daemon with custom sleep and instruction
afk --daemon --sleep 120s --beads --beads-instruction "Work on this. Run tests before closing."
```

---

## 5. Behavior

### 5.1 Max-Iterations Mode

1. Validate inputs. Fail fast on errors.
2. Log session start.
3. **Loop** (1 to N):
   a. If beads active: query `bd ready --json`. No issues → log "no work remaining", exit 0.
   b. Pick first issue (if beads), assemble prompt.
   c. Log iteration start (number, issue ID if applicable).
   d. Invoke agent harness.
   e. Log iteration end (exit code, duration).
   f. Agent failure does **not** stop the loop — logged and continues.
4. Max reached → exit 0.

### 5.2 Daemon Mode

1. Validate inputs. Fail fast on errors.
2. Log session start.
3. **Loop** (indefinitely):
   a. If beads active: query `bd ready --json`. No issues → log "sleeping", sleep, retry.
   b. If beads not active: run prompt, sleep, repeat.
   c. When work found: invoke agent, log results.
   d. After agent completes: immediately check for more work (no sleep between available items).
4. SIGINT/SIGTERM → let current agent finish, then exit cleanly.
5. Second SIGINT → force quit immediately.

### 5.3 Prompt Assembly

Final prompt sent to the agent, in order:

1. **User prompt** (from `--prompt` or stdin), if provided
2. **Issue block** (delimiter + issue JSON from beads), if available
3. **Instruction text** (standard or overridden), after the issue

If only issue (no user prompt): issue + instruction text.
If only user prompt (no beads): user prompt alone.

### 5.4 Issue Selection

Pick the **first issue** from `bd ready --json` output. `bd` returns issues in priority order. `afk` does not re-sort.

---

## 6. Logging

### Session Log File

One file per invocation: `~/.local/share/afk/logs/afk-2026-03-13T14-30-00.log`

### Log Format

```
2026-03-13T14:30:00Z [afk] session-start mode=max-iterations max=20 harness=claude beads=true labels=backend,p0
2026-03-13T14:30:01Z [afk] iteration-start iteration=1 issue=afk-42 title="Fix auth bug"
2026-03-13T14:35:12Z [afk] iteration-end iteration=1 exit-code=0 duration=311s
2026-03-13T14:40:01Z [afk] beads-check count=0
2026-03-13T14:40:01Z [afk] session-end reason="no work remaining" total-iterations=2 duration=600s
```

Daemon mode additionally:
```
2026-03-13T14:40:01Z [afk] sleeping duration=60s
2026-03-13T14:41:01Z [afk] waking
```

### Terminal Output

Default — minimal summary to stdout:
```
afk: starting (max-iterations=20, harness=claude, beads=on)
afk: [1/20] working on afk-42 "Fix auth bug"
afk: [2/20] working on afk-43 "Add tests for parser"
afk: done (2 iterations, no work remaining)
```

`--stderr`: full log mirrored to stderr (watch + file simultaneously).
`--quiet`: no terminal output except errors.
`--verbose`: agent stdout/stderr also shown and captured.

---

## 7. Error Handling

### Fail-Fast (before loop starts)

| Condition | Message |
|---|---|
| `--daemon` + `--max-iterations` | `error: --daemon and --max-iterations are mutually exclusive` |
| `--raw` + `--harness`/`--model`/`--agent-flags` | `error: --raw cannot be combined with --harness, --model, or --agent-flags` |
| `--quiet` + `--stderr`/`--verbose` | `error: --quiet cannot be combined with --stderr or --verbose` |
| No prompt and no beads | `error: no prompt provided and beads integration is not active; nothing to do` |
| Harness binary not in PATH | `error: harness "codex" not found in PATH; is it installed?` |
| `bd` not in PATH when beads active | `error: "bd" not found in PATH; beads integration requires bd to be installed` |
| `--sleep` without `--daemon` | `error: --sleep requires --daemon mode` |
| Log dir not writable | `error: cannot write to log directory "/path": permission denied` |

### Runtime (non-fatal, loop continues)

- **Agent exits non-zero**: logged, loop continues.
- **`bd ready` returns bad JSON or errors**: logged as warning. Treated as "no work" (exit in max-iterations, sleep in daemon).

### Exit Codes

| Code | Meaning |
|---|---|
| 0 | Clean exit (max reached, no work, or signal shutdown) |
| 1 | Runtime error (missing binary, all iterations failed) |
| 2 | CLI usage error (invalid flags, bad arguments) |

---

## 8. Future Considerations (Out of Scope for v1)

- **Parallel loops**: multiple agents working concurrently on different issues
- **Post-run hooks**: user-defined validation after each iteration
- **Config file**: `.afk.yaml` for per-project defaults
- **TUI mode**: live-updating terminal dashboard per iteration
- **Issue selection strategies**: beyond "pick first"
- **Agent timeout**: detecting stuck agents
- **Notifications**: desktop/webhook on session completion
- **Session resume**: re-read log, skip completed iterations
