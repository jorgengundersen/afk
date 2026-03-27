# Harnesses

A harness wraps an external agent CLI so afk can invoke it non-interactively
and, when possible, parse its structured output for consistent terminal
rendering.

**Key principle:** afk's responsibility per harness is exactly two things:

1. Invoke the agent in non-interactive mode (no TTY, no human prompts).
2. Enable structured output so afk can parse and render agent activity.

Everything else — sandbox policy, approval mode, permissions, model provider
configuration — is your responsibility. Configure harness behaviour via
`--harness-args` or the agent's own config files.

**Working directory:** afk does not change the working directory for
harnesses. Run afk in the directory you want the agent to operate on.
Harnesses inherit the current working directory.

## Claude (default)

```sh
afk -p "refactor auth"
afk -p "refactor auth" --model sonnet
afk -p "refactor auth" --harness-args "--dangerously-skip-permissions"
```

- **Binary:** `claude`
- **Invocation:** `claude -p "<prompt>" --output-format stream-json [--model <model>] [harness-args...]`
- **Output:** Structured JSON stream, parsed and rendered by afk
- **Model:** passed via `--model` when set

## Codex

```sh
afk -p "add tests" --harness codex
afk -p "add tests" --harness codex --model o3
afk -p "add tests" --harness codex --harness-args "--full-auto"
```

- **Binary:** `codex`
- **Invocation:** `codex exec "<prompt>" --json [--model <model>] [harness-args...]`
- **Output:** NDJSON event stream, parsed and rendered by afk
- **Model:** passed via `--model` when set
- **Note:** The prompt is a positional argument to `exec`, not a `-p` flag

## OpenCode

```sh
afk -p "fix linting errors" --harness opencode
```

- **Binary:** `opencode`
- **Invocation:** `opencode -p "<prompt>" [--model <model>] [harness-args...]`
- **Output:** Stdout and stderr passed through to your terminal (no structured parsing)
- **Model:** passed via `--model` when set

## Raw (escape hatch)

```sh
afk -p "update changelog" --raw "my-agent {prompt}"
```

- **Invocation:** `sh -c` with the prompt shell-escaped and substituted into the template
- **Output:** Stdout and stderr passed through to your terminal
- **`--harness-args`:** Ignored (you control the full command template)
- **`--model`:** Cannot be used with `--raw`

## What you configure

afk does not inject, default, or override any harness-specific behavioural
flags. You are responsible for:

- **Sandbox policy** — e.g., Claude's `--dangerously-skip-permissions`
- **Approval mode** — e.g., Codex's `--full-auto`
- **Permissions and access** — API keys, tool permissions, etc.
- **Model provider config** — provider-specific environment variables

Pass these via `--harness-args` or configure them in the agent's own config
files.
