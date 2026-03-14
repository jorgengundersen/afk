# Technical Architecture

## Constraint

Go standard library only. No external dependencies.

## Package Layout

```
afk/
├── cmd/
│   └── afk/
│       └── main.go              # Entrypoint: parse args, wire dependencies, run
├── internal/
│   ├── cli/                     # CLI parsing, validation, exit codes, signal setup
│   │   └── cli.go               # Flag definitions, mutual-exclusion checks, fail-fast validation
│   ├── config/                  # Configuration types
│   │   └── config.go            # Validated, immutable config struct built from CLI flags
│   ├── harness/                 # Agent harness abstraction
│   │   ├── harness.go           # Harness interface + New() factory
│   │   ├── claude.go            # Claude Code harness
│   │   ├── opencode.go          # OpenCode harness
│   │   ├── codex.go             # Codex harness
│   │   ├── copilot.go           # Copilot harness
│   │   └── raw.go               # Raw command harness
│   ├── prompt/                  # Prompt assembly
│   │   └── prompt.go            # Build final prompt from user input + beads issue + instruction
│   ├── beads/                   # Beads integration (bd CLI wrapper)
│   │   └── beads.go             # Query bd ready, parse JSON, issue selection
│   ├── loop/                    # Core loop orchestration
│   │   ├── maxiter.go           # Max-iterations mode
│   │   └── daemon.go            # Daemon mode (sleep/wake, signal handling)
│   └── eventlog/                # Structured logging
│       └── eventlog.go          # Logger: file + optional stderr, structured key=value format
```

## Design Decisions

### Packages map to domain concepts, not technical layers

Each package represents one domain concept (harness, prompt, beads, loop, eventlog). No `utils/`, `helpers/`, or `common/` packages.

### Dependency direction flows inward

`cmd/afk/main.go` → `internal/cli/` → `internal/config/` → everything else. Domain packages (`internal/harness/`, `internal/prompt/`, `internal/beads/`, `internal/eventlog/`) do not depend on each other. `internal/loop/` depends on `internal/harness/`, `internal/prompt/`, `internal/beads/`, and `internal/eventlog/` — it is the orchestration layer.

```
cmd/afk/main.go
  └── internal/cli/         (parses flags, builds config)
  └── internal/loop/        (orchestration — the only package that sequences work)
        ├── internal/harness/   (invokes agent)
        ├── internal/prompt/    (assembles prompt)
        ├── internal/beads/     (queries bd)
        └── internal/eventlog/  (writes events)
```

### Orchestration lives in `internal/loop/`, not `main.go`

`cmd/afk/main.go` parses, validates, wires. `internal/loop/` decides iteration logic, sleep/wake, signal response. Clear separation: `main.go` is plumbing, `internal/loop/` is the brain.

### Harness as interface

```go
type Harness interface {
    Run(ctx context.Context, prompt string) (exitCode int, err error)
}
```

Each harness implementation translates this contract into the specific CLI invocation. `raw.go` replaces `{prompt}` in the command string. All harnesses use `os/exec` to run the external binary.

### Config is immutable after construction

`internal/cli/` builds a `config.Config` value from CLI flags. It is passed by value (or pointer-to-immutable) to all packages. No mutation after construction. No global config. No `init()`.

### State ownership

Mutable state exists in exactly two places:
1. **Loop state** (current iteration, running agent process) — owned by `internal/loop/`.
2. **Logger state** (open file handle) — owned by `internal/eventlog/`.

Everything else is stateless: `config` is immutable, `prompt` is a pure function, `harness` implementations hold no state between calls, `beads` queries are stateless.

## External Process Execution

All external process execution (`claude`, `codex`, `bd`, etc.) goes through `os/exec.CommandContext`. Context carries cancellation from signal handling. The harness does not interpret agent stdout/stderr — it captures or forwards based on verbose flag.

## Signal Flow

```
SIGINT/SIGTERM received
  → internal/cli/ notifies internal/loop/ via context cancellation
  → internal/loop/ waits for current harness.Run() to complete
  → internal/loop/ logs session-end, exits 0

Second SIGINT
  → internal/cli/ cancels context with force
  → os.Exit(1)
```
