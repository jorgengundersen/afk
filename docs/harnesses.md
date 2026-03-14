# Harnesses

## Overview

afk delegates work to external agent CLIs through a **harness** abstraction. A harness wraps a specific agent binary, translating afk's prompt into the correct CLI invocation. This keeps the core loop agent-agnostic — swap the harness, swap the agent.

Select a harness with `--harness`:

```
afk --harness claude -p "refactor auth"
afk --harness opencode -p "add tests"
```

The default harness is `claude`.

## Claude (default)

Binary: `claude` (Claude Code CLI)

Invocation:

```
claude -p <prompt> --dangerously-skip-permissions [agent-flags...]
```

The `--dangerously-skip-permissions` flag is always passed so Claude Code runs non-interactively without prompting for tool approvals.

- `--model` is forwarded via `--agent-flags` (e.g. `--agent-flags "--model opus"`)
- `--agent-flags` appends arbitrary flags after the base invocation

Example:

```
afk --harness claude --agent-flags "--model opus" -p "fix the login bug"
```

This runs: `claude -p "fix the login bug" --dangerously-skip-permissions --model opus`

## OpenCode

Binary: `opencode`

Invocation:

```
opencode -p <prompt> --yes [agent-flags...]
```

The `--yes` flag is always passed for non-interactive execution.

- `--model` is forwarded via `--agent-flags` (e.g. `--agent-flags "--model gpt-4o"`)
- `--agent-flags` appends arbitrary flags after the base invocation

Example:

```
afk --harness opencode --agent-flags "--model gpt-4o" -p "add unit tests"
```

This runs: `opencode -p "add unit tests" --yes --model gpt-4o`

## Codex and Copilot (coming soon)

The following harnesses are registered in configuration but not yet implemented:

- **codex** — OpenAI Codex CLI
- **copilot** — GitHub Copilot CLI

These are planned for a future release. Attempting to use them today will fail with a "binary not found" error.

## Raw Command Mode

Use `--raw` to bypass the harness system entirely and run an arbitrary command. The placeholder `{prompt}` in the command string is replaced with the assembled prompt.

```
afk --raw "aider --yes --message {prompt}" -p "fix the login bug"
```

This is useful for agents that afk doesn't have a built-in harness for, or for custom scripts.

More examples:

```
afk --raw "my-custom-agent --input {prompt}" -p "review code"
afk --raw "aider --yes --message {prompt} --model gpt-4o" -p "add error handling"
```

**Restrictions:** `--raw` cannot be combined with `--harness`, `--model`, or `--agent-flags`. These flags are harness-specific and have no meaning in raw mode.

## AGENTS.md

Place an `AGENTS.md` file in your repository root to steer agent behavior. Agents like Claude Code read this file automatically, so it applies within the afk loop — coding standards, testing requirements, commit conventions, and project-specific instructions.

This works with any harness that respects `AGENTS.md` (currently Claude Code). For other agents, use their equivalent configuration mechanism.
