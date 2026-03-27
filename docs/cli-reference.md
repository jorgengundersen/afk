# CLI Reference

## Usage

```
afk [flags]
afk quickstart
```

The `quickstart` subcommand prints a cheatsheet to stdout and exits.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-p` | string | | Prompt text or path to a prompt file |
| `-n` | int | 20 | Maximum iterations before exiting |
| `-d` | bool | false | Daemon mode: run indefinitely |
| `--sleep` | duration | 60s | Sleep duration between daemon cycles |
| `--harness` | string | claude | Agent harness to invoke |
| `--model` | string | | Model override passed to the harness |
| `--raw` | string | | Raw command template with `{prompt}` placeholder |
| `--harness-args` | string | | Additional flags passed through to the harness CLI |
| `--beads` | bool | false | Pull work from beads (`bd ready`) |
| `--label` | string | | Label filter, AND logic (repeatable) |
| `--label-any` | string | | Label filter, OR logic (repeatable) |

## Prompt Sources

A prompt must come from exactly one source (unless `--beads` supplies it):

1. **`-p "text"`** — inline prompt string
2. **`-p file.md`** — read prompt from file (if the argument is a readable file path)
3. **stdin** — piped input (`echo "fix tests" | afk`)
4. **`--beads`** — prompt derived from the next ready bead

## Mutual Exclusion Rules

- `--raw` cannot be combined with `--harness` or `--model`
- `--sleep` requires `-d` (daemon mode)
- `--label` and `--label-any` require `--beads`
- Without `--beads`, a prompt (`-p` or stdin) is required

## Valid Harness Values

| Value | Agent | Structured Output |
|-------|-------|-------------------|
| `claude` | Claude Code | Yes (stream-json) |
| `codex` | OpenAI Codex | Yes (NDJSON) |
| `opencode` | OpenCode | No (stdout passthrough) |

The `--raw` flag bypasses named harnesses entirely. It runs `sh -c` with
the prompt shell-escaped and substituted into the template.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Clean exit |
| 1 | Runtime error or all iterations failed |
| 2 | CLI usage or validation error |
