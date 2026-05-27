# afk

`afk` is a small, general-purpose command loop runner for Unix-like systems (Linux/macOS).

## Status

Implemented and usable.

## Specs

- [Specs index](specs/index.html)
- [PRD: Simplified afk Loop Utility](specs/prd-afk.html)
- [Design Principles](specs/design-principles.html)
- [Architecture](specs/architecture.html)
- [Coding and Testing Standards](specs/coding-testing-standards.html)

## Usage

```text
afk [flags] -- command [args...]
```

`afk` requires at least one loop driver flag:

- `-n`, `--loops <N>`: run exactly N times
- `-d`, `--daemon`: run until interrupted
- `--items <text>`: static item batch (newline text or JSON array)
- `--items-cmd <shell command>`: dynamic item-source command

Other runtime flags:

- `--sleep <duration>`: sleep only after an empty `--items-cmd` batch
- `--empty-sleeps <N>`: non-daemon retries for empty `--items-cmd` batches
- `--fail continue|stop`: main child failure policy (default: `continue`)
- `--until-success`: keep trying until main child returns `0` (with `--loops` or `--daemon`; cannot be combined with explicit `--fail`)
- `--timeout <duration>`: per-process-group timeout for main child and `--items-cmd`
- `-h`, `--help`: print help and exit `0`

Examples:

```bash
# run a command 3 times
afk -n 3 -- sh -c 'echo "attempt $AFK_INDEX"'

# process a static item list
afk --items $'a\nb' -- sh -c 'echo "$AFK_INDEX $AFK_ITEM"'

# daemon mode with dynamic item source
afk -d --items-cmd './list-work' --sleep 30s -- ./worker
```

## Runtime environment variables

`afk` passes loop context to the main child via environment variables:

- `AFK_INDEX`: 1-based global main-child invocation number (always set)
- `AFK_ITEM`: current item value (item-driven loops only)
- `AFK_ITEM_INDEX`: 0-based index within current item batch (item-driven loops only)
- `AFK_ITEM_COUNT`: total item count in current batch (item-driven loops only)

Environment construction behavior:

- Main-child invocations inherit the parent environment, then apply `AFK_*` loop context.
- `AFK_INDEX` is always overwritten by `afk`.
- In item-driven loops, `AFK_ITEM`, `AFK_ITEM_INDEX`, and `AFK_ITEM_COUNT` are overwritten.
- In non-item loops, inherited `AFK_ITEM`, `AFK_ITEM_INDEX`, and `AFK_ITEM_COUNT` are removed.

## `--items-cmd` trust and shell model

`--items-cmd` is executed as `/bin/sh -c <value>`.

- Treat `--items-cmd` as shell code, not as an argv-safe command list.
- Do not build `--items-cmd` from untrusted interpolated input.
- Each `--items-cmd` invocation captures at most 8 MiB of stdout for item parsing; if stdout exceeds that limit, `afk` discards the captured stdout from that invocation and exits with source error code `1`.
- Each `--items-cmd` invocation inherits the parent environment with `AFK_INDEX`, `AFK_ITEM`, `AFK_ITEM_INDEX`, and `AFK_ITEM_COUNT` removed.
- `--items-cmd` receives no loop context because it runs before the next main-child invocation exists.

## Build and test

```bash
go test ./...
go build ./cmd/afk
```

## Project analysis workflow assets

Tracked analysis assets:

- Prompt: `docs/project-analysis/PROJECT-ANALYSIS.md`
- Artifact contract: `docs/project-analysis/PROJECT-ANALYSIS-ARTIFACTS.md`

Run-scoped exploratory outputs belong under `tmp/project-analysis/<run-id>/`.
That workspace is intentionally temporary and gitignored.
