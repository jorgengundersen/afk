# Quality Guard

Pre-commit quality gate for agentic coding. Defines what runs before every
commit, why it runs, and the contract between agents and the guardrails.

This project is developed primarily by AI agents. Agents will generate code
at speed — the quality guard is the automated check that prevents broken,
unformatted, or untested code from entering the commit history.

## Tool: Lefthook

[Lefthook](https://github.com/evilmartians/lefthook) orchestrates all git
hooks. It is the single owner of `.git/hooks/`. Beads integrates through
lefthook — not by installing its own hook shims.

**Setup:** `lefthook install` (one-time, after clone).

**Config:** `lefthook.yml` at repo root.

## Pre-commit Checks

All checks run in parallel. A failure in any check blocks the commit.

### Go Checks (conditional)

These checks only run when `.go` files are staged. When no Go source exists
(e.g., during spec-writing phases), they are skipped automatically via
lefthook's `glob` filter.

| Check | Command          | What it catches                    |
|-------|------------------|------------------------------------|
| `fmt` | `gofmt -l .`     | Unformatted code                   |
| `vet` | `go vet ./...`   | Suspicious constructs, type errors |
| `test`| `go test ./...`  | Test failures, compilation errors  |

**Why these three?** Each catches a different class of defect:
- `fmt` is cosmetic but prevents noisy diffs and style arguments.
- `vet` catches bugs that compile but are almost certainly wrong.
- `test` compiles and runs all tests. This subsumes `go build` — if it
  compiles for test, it compiles. The race detector (`-race`) is omitted
  from pre-commit because it requires CGO/gcc; use it in CI instead.

### Beads Hook

| Check         | Command                    | What it does                 |
|---------------|----------------------------|------------------------------|
| `beads-hooks` | `bd hooks run pre-commit`  | Syncs issue tracker state    |

Runs unconditionally. The `BD_GIT_HOOK=1` env var tells beads it's running
inside a git hook context.

## Contract

1. **No bypass without reason.** Agents must not use `--no-verify`. If a
   hook fails, the agent must fix the issue, not skip the check.

2. **All checks must pass before commit.** A commit with failing checks is
   a broken commit. The cost of fixing later always exceeds fixing now.

3. **Checks must be fast.** Pre-commit checks run on every commit. If a
   check takes more than 30 seconds on the full codebase, it should move
   to CI or be scoped to changed files.

4. **New tools extend, not replace.** When adding a linter or checker, add
   it as a new lefthook command. Do not wrap checks in custom scripts or
   create parallel hook systems.

## Emergency Bypass

In rare cases where a commit must land despite a failing check (e.g., a
check is broken and blocking unrelated work), use:

```bash
git commit --no-verify -m "reason for bypass"
```

File a bug immediately for the broken check. This is an escape hatch, not
a workflow.

## Adding New Checks

1. Add the command to `lefthook.yml` under `pre-commit.commands`.
2. Use `glob` to scope it to relevant file types.
3. Update this spec with the new check.
4. Test on a throwaway branch before merging.
