---
name: create-issue-graph
description: Plan and file a graph of beads (bd) issues for havn — epics, tasks, dependencies, parent/child links. Use when decomposing an epic, turning a research report into issues, filing bug reports, or adding tasks under an existing epic. Always shows a reviewable plan first, then files via parallel `bd create` + `bd dep add` only after user confirmation.
---

# create-issue-graph

Plan → show → file on confirmation.

## Phase 1 — PLAN (always)

Show a markdown plan in chat. Do **not** file yet.

```markdown
## Planned graph

| Ref | Title | Type | Prio | Parent | Blocks | Notes |
|---|---|---|---|---|---|---|
| n1 | … | task | P2 | havn-xxx | — | — |
| n2 | … | task | P2 | havn-xxx | n1 | — |

### Draft descriptions
**n1 — <title>**
<havn template>

Ready to file? (yes / revise / cancel)
```

Before showing the plan, mentally lint each leaf:
- observable behavior or tight contract?
- acceptance stated behaviorally?
- no separate test task required?
- not mostly scaffolding or setup?
- small enough for one agent context window?

If a leaf fails this check, rewrite it before presenting the plan.

`revise` → update + reconfirm. `cancel` → abort. `yes` → Phase 2.

## Phase 2 — FILE (only after confirm)

1. Create in dependency order. Parents before children.
2. Parallelize only across different parents. Under the same parent, create serially.
3. Add dependencies after creates succeed.
4. Verify with `bd list` and `bd ready -n 100`.
5. Report created IDs and the resulting ready shape.

## Core rules

### 1. Priority = urgency, not ordering
- Start at **P2**.
- Use **P0** only for critical breakage.
- Use **P1** only when it must preempt current work immediately.
- Use **P3/P4** for lower-urgency or deferred work.
- Never use priority to serialize sibling tasks.

### 2. Dependencies model real blockers only
- `blocks` means B cannot make meaningful progress until A lands.
- `--parent` is containment, not ordering.
- Keep siblings parallel unless one truly blocks the other.
- Put cross-graph blockers on the specific leaf that needs them, not on the epic, unless every child needs the prereq.
- Task → epic blocking is the wrong model; block the relevant child task instead.
- Closing all children auto-closes the epic; do not add “epic blocked by children” deps.

### 3. Leaves must be behavior slices or tight contract slices
- One cohesive behavior, small enough for one agent context window.
- Aim for a vertical slice, roughly 2–5 TDD cycles.
- Each leaf must have an observable outcome:
  - user-visible behavior, or
  - CLI/API contract, or
  - pure-function contract with clear inputs/outputs.
- If a leaf mainly creates files, directories, types, wiring, or scaffolding, merge it with the first behavior that proves that setup is useful.
- For greenfield work, the first leaf should usually be buildable/runnable plus one observable behavior.
- Never split work into design/impl/test phases. Never create separate “write tests” tasks.

Examples:
- ✅ `Config.Load reads global TOML and handles missing/malformed input`
- ✅ `Build runnable CLI with --help output`
- ❌ `Add Config.Load signature`
- ❌ `Bootstrap module and entrypoint` (unless paired with concrete behavior)
- ❌ `Implement config package` (epic)

### 4. Acceptance = behavior, not mechanics
Describe what changes for the user or contract.
- ✅ ``afk --help`` prints usage to stdout and exits `0`
- ✅ ``havn list --json`` emits the documented shape
- ❌ `Table-driven tests cover edge cases`
- ❌ `Add parser package and wire main`

### 5. Issue types and non-blocking links
- `epic` — multi-task container
- `feature` — user-visible functionality
- `task` — internal work; default for children
- `bug` — broken behavior
- `chore` — tooling, deps, CI

Use non-blocking deps only when accurate:
- `discovered-from` — literally found while doing linked work
- `related` — informational
- `caused-by` — known root cause
- `supersedes` — replacement

## havn description template

```markdown
<one-paragraph summary + why>

## Scope
- <what this covers>

## Out of scope
- <only if ambiguous>

## Specs
- specs/<f>.md §<n>

## Wires up
- <what connects when this closes; domain-logic only>

## Acceptance
- <user-visible behavior or contract; not file layout or test mechanics>
```

## Filing mechanics

```bash
bd create "<title>" \
  --type <epic|task|feature|bug|chore> \
  --priority <0-4> \
  --parent <id> \
  --spec-id <spec-name> \
  --description "<template>" \
  --json

bd dep add <dependent> <prereq>             # default: blocks
bd dep add <from> <to> --type discovered-from
bd dep add <from> <to> --type related

bd list
bd ready -n 100
```

## Hard rules

- Never file before user confirms.
- Never claim work (`bd update --claim`) — that's for the implementation agent.
- Never update existing issues via this skill — use `bd update` directly.
- Always `bd search` / `bd list` before filing to avoid duplicates.
- Use `bd create` + `bd dep add`; do not batch-file via undocumented graph shortcuts.
