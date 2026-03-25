---
name: create-epic
description: Decompose spec/feature/refactor into epic(s) with ordered children in beads. Use when work spans multiple commits.
argument-hint: <spec-path-or-description>
allowed-tools: Bash(bd *), Read, Glob, Grep
---

Decompose $ARGUMENTS into epic(s) with child tasks.

## Epic = single goal, multiple commits

All children serve one goal. When all closed, epic is done.

**One or many?** If "AND" joins two goals that ship independently — two epics.
If >10 children — break into sub-epics:

```bash
# Top-level epic
bd create "Auth system" -t epic -p 1 -d "Full auth implementation" --json
# → bd-a1

# Sub-epics under it
bd create "Login flow" -t epic -p 1 --parent bd-a1 -d "Login UI + validation" --json
# → bd-a1.1
bd create "Session mgmt" -t epic -p 1 --parent bd-a1 -d "Token lifecycle" --deps bd-a1.1 --json
# → bd-a1.2

# Tasks under sub-epic
bd create "Login form" -t task --parent bd-a1.1 -d "Build login component" --json
# → bd-a1.1.1
```

## Ordering children

`--deps` encodes serial/parallel. `bd ready` uses it.

```
Serial:    A → B → C          (--deps chains)
Parallel:  A   B   C          (no deps between them)
Diamond:   A → C, B → C       (C: --deps A,B)
```

Default serial. Only parallelize when clearly independent — wrong parallelism
causes conflicts, wrong serialization just slows down.

## TDD-compatible task structure

This project uses red/green TDD. The implementing agent handles the TDD
procedure — epic tasks don't prescribe it. Instead, scope each code task as
a single testable behaviour so TDD applies naturally.

**Task descriptions should provide:**
- What to build (the behaviour/contract)
- Why it exists (how it fits in the larger picture)
- Constraints (design decisions, boundaries, what's in/out of scope)

**Task descriptions should NOT prescribe:**
- Specific test cases
- RED/GREEN labels or implementation steps

The implementing agent has the spec, the codebase, and TDD discipline to
decide test cases and implementation order.

Example:

```bash
bd create "ParseFlags function" -t task --parent bd-x \
  -d "ParseFlags(args []string) (Config, error) using local FlagSet. Defaults: MaxIter=20, Sleep=60s, Harness=claude. Track explicitly-set flags via FlagSet.Visit for later validation." --json
```

## Process

1. Read spec/description.
2. Apply single-goal test — one epic or graph of epics.
3. Create epic: `bd create "<goal>" -t epic -p <0-3> -d "<scope>" --spec-id <path> --json`
4. Create children (each = one testable commit):
   ```bash
   bd create "<task>" -t task --parent <epic> -d "<what+done>" --json
   bd create "<task>" -t task --parent <epic> -d "<what+done>" --deps <prev> --json
   ```
   For code tasks, scope each as a single testable behaviour.
5. Verify: `bd dep tree <epic> --json`
6. Show dep tree to user. No implementation until approved.
