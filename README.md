# afk

Run agentic coding loops unattended. Type `afk`, walk away, come back to completed work.

## Install

Requires Go 1.22+ and at least one supported agent CLI (see below).

```sh
go install github.com/jorgengundersen/afk/cmd/afk@latest
```

You also need a supported agent binary on your `$PATH`. The default is
[Claude Code](https://docs.anthropic.com/en/docs/claude-code) (`claude`).

## Quick Start

Run a one-shot prompt for up to 20 iterations (the default):

```sh
afk -p "refactor the auth middleware to use JWT"
```

Feed a prompt from a file:

```sh
afk --prompt instructions.md
```

Run as a daemon, pulling work from beads and sleeping between cycles:

```sh
afk -d --beads --sleep 5m
```

Use a different agent:

```sh
afk -p "add input validation" --harness opencode
```

Pipe a prompt from stdin:

```sh
echo "fix the failing tests" | afk
```

Limit to a single iteration:

```sh
afk -n 1 -p "update the changelog"
```

## How It Works

afk wraps an agent CLI in a retry loop. Each iteration assembles a prompt,
invokes the agent, and records structured logs. In **max-iterations** mode
(the default) it runs up to `-n` iterations and exits. In **daemon** mode
(`-d`) it runs indefinitely, sleeping between cycles.

When `--beads` is enabled, afk queries `bd ready --json` for open issues at
the start of each iteration. It claims the highest-priority bead, feeds it
to the agent as part of the prompt, and closes the bead when the agent
finishes. This turns afk into a self-directing work loop — no human
prompting required.

## Supported Agents

| Agent | Flag | Status |
|-------|------|--------|
| Claude Code | `--harness claude` (default) | Implemented |
| OpenCode | `--harness opencode` | Implemented |
| OpenAI Codex | `--harness codex` | Planned |
| GitHub Copilot | `--harness copilot` | Planned |
| Any CLI | `--raw "cmd {prompt}"` | Implemented |

The `--raw` flag is an escape hatch: pass any command string with a
`{prompt}` placeholder and afk will substitute the prompt at runtime.

## Documentation

| Document | Description |
|----------|-------------|
| [CLI Reference](docs/cli-reference.md) | All flags, mutual-exclusion rules, exit codes |
| [User Guide](docs/user-guide.md) | Concepts, workflows, tips |
| [Harnesses](docs/harnesses.md) | Agent invocation details, AGENTS.md |
| [Logging](docs/logging.md) | Structured log format, event types |

## License

[GPL-3.0](LICENSE)
