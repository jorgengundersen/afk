# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.1] - 2026-04-14

### Fixed

- **Nix packaging**: pin flake package builds and dev shell to Go 1.26 (`go_1_26`) so sandboxed builds with `GOTOOLCHAIN=local` satisfy `go.mod` and don't fail on toolchain mismatch.
- **Harness**: disable Go's default `cmd.Cancel` so `runCmd` is the sole cancellation path; prevents a SIGKILL race that could skip graceful SIGTERM shutdown of the agent process group.
- **Signal**: handle double Ctrl+C by running registered force-kill hooks on the second signal; `runCmd` registers a hook that SIGKILLs the agent process group, preventing orphaned agent processes.

### Removed

- **Harness**: non-unix platform support. `runcmd_other.go` (the fallback for non-unix GOOS values) lacked process group management and could not uphold the new signal contract. The harness is now unix-only.

## [2.0.0] - 2026-03-27

Complete rewrite of the Go implementation with a common event model,
Codex harness support, and user documentation.

### Added

- **Common event model**: `CommonEvent` type with `EventKind` discriminator (Text, ToolCall, ToolOutput, Summary) and typed payloads, enabling multi-harness structured output.
- **Codex harness**: first-class Codex support via `codex exec --json` with NDJSON parsing through the common event model.
- **Codex adapter**: `ParseCodexStream` maps Codex events (agent_message, reasoning, command_execution, mcp_tool_call, file_change, web_search, turn.completed/failed) to common events.
- **`afk quickstart`**: cheatsheet subcommand for quick reference.
- **CLI**: `--harness-args` flag for passing additional flags to harness CLIs.
- **Config**: `--label` and `--label-any` repeatable flags for beads label filtering.
- **Beads integration**: `beads.Client` wrapping `bd` CLI, `WorkSource` interface for iteration gating, issue composition in prompt assembly.
- **Harness rendering**: Claude terminal renderer with structured JSON stream parsing.
- **Process groups**: subprocess runs in its own process group; cancellation kills the entire tree.
- **Signal handling**: `NotifyContext` for signal-based context cancellation.
- **Structured logging**: per-session log files with key=value format.
- **User documentation**: CLI reference, user guide, harnesses guide, logging reference.
- **End-to-end tests**: beads happy path and integration tests for process group cleanup.
- **Race detector**: enabled in test suite.
- **Skills**: agent skills for epic creation, bug filing, and human handoff.

### Changed

- **Claude adapter**: refactored to emit `CommonEvent` instead of Claude-specific types. Old wire-format types are now unexported.
- **Renderer**: `RenderStream` accepts `<-chan CommonEvent` instead of `io.Reader`, decoupling rendering from parsing. Summary rendering displays available fields (duration, cost, tokens) rather than assuming Claude's format.
- **README**: updated to reflect Codex as implemented with quick start example.

### Fixed

- **Stream parser**: warns on skipped malformed JSON and unknown events instead of silently dropping them.
- **Loop**: WorkSource errors treated as failed iterations in max-iter mode; daemon mode returns exit code 1 when all iterations fail; iteration counter incremented in daemon mode.
- **Logger**: `Log` returns error; exit immediately on log failure; nil file pointer and closed flag handled in `Close`.
- **Config**: usage written to stdout on `-h` with exit code 0; whitespace-only `--raw` no longer panics.
- **Harness**: fresh pipe created per `Claude.Run()` call; non-zero exit codes included in allFailed check.
- **Loop**: `sync.Mutex` added to test helpers to fix race conditions.

### Removed

- **Legacy Go implementation**: removed old codebase pending rewrite (replaced by current implementation).

## [1.2.0] - 2026-03-20

### Added

- **Harness**: claude harness now passes `--output-format stream-json --verbose` for real-time visibility into agent activity during headless runs.

## [1.1.0] - 2026-03-20

### Added

- **Nix flake** for building and development.

### Fixed

- **Harness**: connect child process stdout/stderr to terminal.
- **Harness**: wire `--model` flag into claude and opencode harnesses.
- **CLI**: prevent goroutine leak in `SetupSignals` on context cancel.
- **Doctor**: propagate errors in `PrintJSON`.
- **Prompt**: propagate JSON unmarshal error in `formatIssue`.
- **Docs**: correct flag dashes and add model value guidance.
- **Logging**: align logging events with spec and fix test fragilities.

## [1.0.0] - 2026-03-18

Initial release of afk — an autonomous loop runner for AI coding agents.

### Added

- **Core loop engine** with two modes: max-iterations (`-n`) and daemon (`-d`) with configurable sleep intervals.
- **Agent harness abstraction** with built-in support for Claude Code and OpenCode, plus a `--raw` escape hatch for arbitrary CLI commands.
- **Beads integration** for autonomous issue processing via `bd ready --json`, with label filtering (`--beads-labels`) and custom instruction text (`--beads-instruction`).
- **Prompt assembly** combining user prompts, beads issues, and agent instructions into a single prompt string.
- **Structured event logging** in key=value format to `~/.local/share/afk/logs/` (configurable via `--log`), with optional stderr mirroring (`--stderr`).
- **Terminal output layer** with human-friendly progress messages, `--quiet` and `-v` verbosity controls.
- **Signal handling** for graceful shutdown on SIGINT/SIGTERM.
- **`afk doctor` subcommand** for environment health checks:
  - Detects known agent harnesses (claude, opencode, codex, copilot) in PATH.
  - Checks for beads runtime (`bd` binary) and project directory (`.beads/`).
  - Runs `bd doctor --json` and forwards full diagnostic results.
  - Human-readable output by default, `--json` flag for machine-readable output.
- **CLI flags**: `-n`, `-d`, `--sleep`, `--harness`, `--model`, `--agent-flags`, `--raw`, `--prompt`, `--beads`, `--beads-labels`, `--beads-instruction`, `--log`, `--stderr`, `-v`, `--quiet`, `-C`.
- **Documentation**: user guide, CLI reference, harness documentation, logging reference, and README.
- **End-to-end and unit test suites** covering all packages.

[Unreleased]: https://github.com/jorgengundersen/afk/compare/v2.0.1...HEAD
[2.0.1]: https://github.com/jorgengundersen/afk/compare/v2.0.0...v2.0.1
[2.0.0]: https://github.com/jorgengundersen/afk/compare/v1.2.0...v2.0.0
[1.2.0]: https://github.com/jorgengundersen/afk/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/jorgengundersen/afk/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/jorgengundersen/afk/releases/tag/v1.0.0
