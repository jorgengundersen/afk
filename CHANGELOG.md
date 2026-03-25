# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Skills**: agent skills for epic creation (`create-epic`), bug filing (`file-bug`), and human handoff (`needs-human`).
- **Docs**: beads workflow documentation (`docs/beads-workflows.md`).
- **Specs**: new project specs — `afk-overview.md`, `design-principles.md`, `skeleton.md`.

### Removed

- **Go implementation**: removed legacy Go codebase (`cmd/`, `internal/`, `e2e/`), old specs, and related docs pending rewrite.

### Changed

- **gitignore**: added `tmp/` and beads credential key exclusions.

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

[Unreleased]: https://github.com/jorgengundersen/afk/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/jorgengundersen/afk/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/jorgengundersen/afk/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/jorgengundersen/afk/releases/tag/v1.0.0
