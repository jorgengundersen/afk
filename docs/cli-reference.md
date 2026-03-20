# CLI Reference

`afk` runs agentic coding loops unattended. It has no subcommands — all behavior is controlled through flags.

## Synopsis

```
afk [flags] [-- passthrough-args...]
```

Prompts can be provided via `--prompt` or piped through stdin.

## Flags

### Mode

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-n` | int | `20` | Maximum number of loop iterations |
| `-d` | bool | `false` | Daemon mode — loop indefinitely |
| `--sleep` | duration | `60s` | Sleep interval between iterations in daemon mode |

In the default max-iterations mode, afk runs up to `-n` iterations and exits. In daemon mode (`-d`), afk runs indefinitely, sleeping when idle.

`--sleep` only applies in daemon mode. The default sleep interval is 60 seconds.

### Harness

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--harness` | string | `claude` | Agent harness to use: `claude`, `opencode`, `codex`, `copilot` |
| `--model` | string | _(empty)_ | Model name passed to the harness (valid values depend on the harness — see [Harnesses](harnesses.md)) |
| `--agent-flags` | string | _(empty)_ | Extra flags forwarded to the harness CLI |
| `--raw` | string | _(empty)_ | Raw command string (bypasses harness) |

The `--harness` flag selects which agent CLI to invoke. Currently implemented harnesses:

- **claude** — Claude Code (default)
- **opencode** — OpenCode

The following harnesses are registered but not yet implemented — planned for a future release (coming soon):

- **codex** — OpenAI Codex CLI
- **copilot** — GitHub Copilot CLI

Use `--raw` to bypass the harness system entirely and run an arbitrary command. The placeholder `{prompt}` in the command string is replaced with the assembled prompt.

### Prompt

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--prompt` | string | _(empty)_ | Prompt text for the agent |

If `--prompt` is not set, afk reads from stdin (when piped). At least one of `--prompt`, stdin, or `--beads` must provide input — otherwise afk exits with an error.

### Beads

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--beads` | bool | `false` | Enable beads integration |
| `--beads-labels` | string | _(empty)_ | Comma-separated label filter (implies `--beads`) |
| `--beads-instruction` | string | _(default text)_ | Override the instruction text appended to each issue |

When `--beads` is active, afk queries `bd ready --json` each iteration to pick the next issue. Setting `--beads-labels` implies `--beads`.

### Logging

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--log` | string | `~/.local/share/afk/logs/` | Directory for session log files |
| `--stderr` | bool | `false` | Mirror log output to stderr |
| `-v` | bool | `false` | Increased verbosity |
| `--quiet` | bool | `false` | Suppress all output except errors |

### Other

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-C` | string | _(empty)_ | Change to directory before running |

Arguments after `--` are passed through to the harness command.

## Mutual Exclusion Rules

Certain flag combinations are invalid and cause afk to exit immediately with exit code 2:

- `--raw` cannot be combined with `--harness`, `--model`, or `--agent-flags`
- `--quiet` cannot be combined with `--stderr` or `-v`
- `--sleep` requires `-d` (daemon mode)
- No prompt and no `--beads` — afk requires at least one source of work

## Exit Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | Clean | Successful execution |
| 1 | Runtime error | An error occurred during execution |
| 2 | Usage error | Invalid flags or flag combinations |

## Examples

### Run a single prompt for up to 5 iterations

```
afk --prompt "refactor the auth module" -n 5
```

### Daemon mode with beads integration

```
afk -d --beads
```

### Filter beads issues by label

```
afk --beads-labels "backend,p0" --prompt "fix the highest priority issue"
```

### Custom sleep interval in daemon mode

```
afk -d --beads --sleep 5m
```

### Pipe a prompt from stdin

```
echo "fix all TODO comments" | afk -n 10
```

### Use a different harness and model

```
afk --harness opencode --model gpt-4o --prompt "add unit tests"
```

### Raw command mode with aider

```
afk --raw "aider --yes --message {prompt}" --prompt "fix the login bug"
```

### Run in a different directory

```
afk -C ~/projects/other-repo --beads -d
```

### Verbose output for debugging

```
afk --beads -v -n 1
```

### Quiet mode for background operation

```
afk --beads -d --quiet
```
