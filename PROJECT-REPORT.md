# Project Report: Post-Implementation Review

**Date:** 2026-03-25
**Epic:** afk-utv

## Summary

All six implementation epics (config, prompt-assembly, harness, event-logger, loop, signal) have been reviewed across four domains: spec alignment, code quality, safety, and test coverage. The codebase is in good shape — implementations closely follow specs, design principles are respected, and test coverage is comprehensive. Five issues were filed during the review; two have already been fixed.

## 1. Spec Alignment (afk-utv.1)

Compared all 6 specs against their implementations.

- **5 of 6 packages** have zero drift between spec and implementation
- **Loop** has minor spec-behind-code drift where the code is correct and the spec should be updated:
  - Function signature uses local interfaces (`Runner`, `Logger`) instead of concrete types — enables testability
  - `session-start` event includes `mode` and `maxIter` fields not in spec — useful for observability
  - Daemon mode uses fixed iteration `0` — spec doesn't specify

**Issues filed:**
| ID | Title | Priority | Status |
|----|-------|----------|--------|
| afk-srg | Update loop spec to match implementation | P3 | Open |

## 2. Code Quality & Design Principles (afk-utv.2)

Reviewed all code in `internal/` and `cmd/` against `specs/design-principles.md`.

- **Principle 1 (Composition):** All packages pass the "and" test. Pure functions used where possible.
- **Principle 3 (Build vs Depend):** Only stdlib dependencies.
- **Principle 4 (Error Handling):** Correct propagation. No log-and-return antipattern.
- **Principle 5 (Dependency Isolation):** One violation — `loop.Run` imports `config.Config` directly.
- **Principle 6 (State):** State pushed to edges. No global state.
- **Principle 7 (Observability):** Structured events at loop edges. Pure functions are silent.
- **Principle 8 (Incremental Complexity):** No premature abstractions.
- **Principle 9 (Codebase is the Prompt):** Consistent patterns, clear naming.

**Issues filed:**
| ID | Title | Priority | Status |
|----|-------|----------|--------|
| afk-5w1 | loop.Run couples to config.Config instead of own types | P3 | Open |

## 3. Safety & Security (afk-utv.3)

Audited all code for safety concerns.

- **Raw shell escaping:** Correct POSIX single-quote technique — no injection risk
- **Logger path handling:** No path traversal concerns
- **Signal handling:** Clean context-based cancellation
- **Resource leaks:** No goroutine leaks; logger close is correct

**Issues filed:**
| ID | Title | Priority | Status |
|----|-------|----------|--------|
| afk-fyh | CheckBinary panics on whitespace-only --raw value | P1 | **Fixed** |
| afk-qqc | Logger.Close does not nil file pointer | P2 | **Fixed** |
| afk-b4w | Design decision: process group management for subprocess cleanup | P2 | Blocked (needs human) |

**Human review required:** afk-b4w — `exec.CommandContext` does not use process groups. If spawned agents create child processes, context cancellation only kills the direct child. Fixing requires platform-specific `SysProcAttr` with `Setpgid`. This is a design decision about scope and platform support.

## 4. Test Coverage (afk-utv.4)

Reviewed all test files for contract verification.

- All spec behaviour rules now have corresponding tests
- 6 missing tests were added during review:
  - SIGTERM signal test
  - Logger lazy open and directory creation tests
  - Loop iteration bracketing, iteration-end fields, and waking event tests
- All tests are black-box (test contracts, not internals)
- Tests are refactoring-resilient

No issues filed — gaps were fixed directly.

## Issues Summary

| ID | Title | Type | Priority | Status | Domain |
|----|-------|------|----------|--------|--------|
| afk-fyh | CheckBinary panics on whitespace-only --raw | Bug | P1 | Fixed | Safety |
| afk-qqc | Logger.Close nil pointer gap | Bug | P2 | Fixed | Safety |
| afk-b4w | Process group management for subprocess cleanup | Task | P2 | Blocked | Safety |
| afk-srg | Update loop spec to match implementation | Task | P3 | Open | Spec alignment |
| afk-5w1 | loop.Run couples to config.Config | Bug | P3 | Open | Code quality |

## Overall Assessment

**Project health: Good.**

The codebase follows its design principles consistently. Critical safety bugs (P1, P2) were identified and fixed during the review. Two low-priority structural items remain open (spec update and dependency isolation refactor). One design decision (process group management) requires human input on scope and platform support trade-offs.
