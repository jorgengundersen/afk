# Coding Standards

## Language

Go. Standard library only. No `go.sum` entries beyond the module itself.

## Formatting and Style

- `gofmt` is law. No exceptions.
- `go vet` must pass with zero findings.
- No build tags unless absolutely necessary.

## Naming

- Packages: short, single-word, lowercase. No underscores. (`harness`, `prompt`, `beads`, `loop`, `eventlog`). Avoid shadowing stdlib package names.
- Exported types/functions: describe what it does in the domain, not how. `Harness`, not `ExternalProcessRunner`. `AssemblePrompt`, not `BuildStringFromParts`.
- Unexported helpers: short names are fine when scope is small. `run`, `parse`, `validate`.
- Avoid stuttering: `harness.Harness` is acceptable (it's the interface), but `harness.HarnessRunner` is not.
- Constants: `ExitClean`, `ExitStartupError`, `ExitAllFailed`. Not `EXIT_CODE_CLEAN`.
- Test files: `*_test.go` in the same package for white-box tests, `*_test.go` with `_test` package suffix for black-box tests. Prefer black-box (see testing standards).

## Package Design

### Exported surface

Every package exports only what other packages need. Default to unexported. Promote to exported only when another package requires it.

### No `init()`

All initialization is explicit in `cmd/afk/main.go`. `init()` is hidden state — violates explicit-over-implicit.

### No global mutable state

No package-level `var` that gets mutated. Config passed as parameter. Logger passed as parameter. No singletons.

### Interfaces at the consumer

Define interfaces where they are used, not where they are implemented. `loop/` defines `Harness` if it's the only consumer. If multiple consumers need the same interface, promote it to its own package.

**Exception — factory functions:** When a package provides a `New()` factory that returns different concrete types (e.g., `harness.New()` returns a Claude, Codex, or Raw harness based on config), returning the interface from the factory is idiomatic. The interface lives in the same package as the implementations it selects between.

## Functions

### Pure by default

If a function can be pure (no I/O, no mutation of external state), make it pure. Prompt assembly is pure. Config validation is pure. Exit code determination is pure.

### Small and focused

A function does one thing. If you're writing a comment to explain "this part does X and then Y", split into two functions.

### Error returns

Every function that can fail returns `error` as the last return value. No panics for expected failures. Panics are reserved for programmer errors (impossible states, violated invariants).

### Context propagation

Functions that do I/O or could be cancelled accept `context.Context` as first parameter. No exceptions.

## Error Handling

See `error-handling.md` for full specification. Summary:

- Errors are values. Use `fmt.Errorf` with `%w` for wrapping.
- Add context at each level: `fmt.Errorf("querying beads: %w", err)`.
- Handle at boundaries (CLI layer, loop orchestrator). Propagate everywhere else.
- Typed error sentinels for conditions callers need to match on.
- Never swallow errors silently.

## File Organization Within a Package

```
package_name/
├── thing.go          # Primary type/interface and its core methods
├── thing_other.go    # Additional implementations or helpers (only if thing.go > ~200 lines)
└── thing_test.go     # Tests
```

One file per major concept. Split when a single file exceeds ~300 lines, but prefer cohesion over arbitrary limits.

## Comments

- Exported symbols get a doc comment. One sentence starting with the symbol name.
- Unexported symbols: comment only when the why isn't obvious.
- No TODO comments in committed code. File an issue instead.
- No commented-out code. Delete it; git has history.

## Dependencies Between Packages

- No circular imports (Go enforces this, but design to avoid needing workarounds).
- No package depends on `cli/`. `cli/` depends on everything.
- `loop/` is the only package that composes other domain packages.
- Domain packages (`harness/`, `prompt/`, `beads/`) are independent of each other.
