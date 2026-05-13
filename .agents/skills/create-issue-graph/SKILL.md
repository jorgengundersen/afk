---
name: create-issue-graph
description: Plan and file a graph of beads (bd) issues for havn — epics, tasks, dependencies, parent/child links. Use when decomposing an epic, turning a research report into issues, filing bug reports, or adding tasks under an existing epic. Always shows a reviewable plan first, then files via parallel `bd create` + `bd dep add` only after user confirmation.
---

# create-issue-graph

Plan → show → file on confirmation.

## Two phases

### Phase 1 — PLAN (always)

Markdown plan in chat. Do NOT file.

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

`revise` → update + reconfirm. `cancel` → abort. `yes` → Phase 2.

### Phase 2 — FILE (only after confirm)

1. Create in dep order. Parents before children. Parallel across *different* parents; serial under *same* parent (Rule 9).
2. `bd create` with full flags.
3. `bd dep add` after creates succeed.
4. Verify: `bd list`, `bd ready -n 100`. Confirm expected leaves ready, nothing unexpectedly blocked.
5. Report filed IDs + `bd ready` shape.

---

## Rules

### 1. Priority = urgency, not ordering
P0 crit · P1 high · **P2 default** · P3 low · P4 backlog. Children inherit parent prio; bump only for genuinely different urgency. Never use prio to serialize.

Priority evaluation checklist:
- Start at **P2**.
- Use **P0** only for critical breakage (security/data loss/broken default workflow/build).
- Use **P1** only when it must preempt current repo focus immediately.
- Use **P3** for non-urgent or non-current-stream improvements.
- Use **P4** for deferred parking-lot ideas.
- If a child differs from parent priority, add a short reason in notes.

### 2. `blocks` = actual blocker only
Add `blocks` only when B literally cannot make meaningful progress until A lands: e.g. B can't compile/run, cannot validate behavior, or truly needs A's artifact/runtime state/contract decision first. If A and B can be worked in parallel, they must not be linked with `blocks`. Never use `blocks` for preference, cleanliness, review order, batching, or phase ordering.

### 3. Cross-epic deps on LEAF, not parent
Bd propagates block state parent → children. Epic-level `blocks` blocks *every* child. Put cross-graph deps on specific leaf that needs the prereq. Epic-level only when *every* child needs it (rare).

### 4. `--parent` = containment, not blocking
Open parent doesn't block children. Blocked parent blocks all children transitively. Sibling ordering needs explicit `blocks` between specific siblings only when one sibling is an actual blocker under Rule 2; otherwise keep them parallel.

### 5. Leaf size = one agent context window
One cohesive behavior. Fits in one context window without derailing. Vertical slice, ~2–5 TDD cycles. Merge if splitting leaves sub-tasks with no observable result on their own (skeleton — anti-pattern 6).
- ✅ *"Config.Load reads global TOML (handles missing and malformed)"*
- ❌ fine: *"Add Config.Load signature"*
- ❌ coarse: *"Implement config package"* (epic)

### 6. No "write tests" sibling tasks
TDD: test drives impl same cycle. Never split impl/test. Never split epic into design/impl/test phases.

### 7. Acceptance = behavior, not test mechanics
- ✅ *"`havn list --json` emits shape in havn-overview.md §3"*
- ✅ *"Stopping missing container returns `container.NotFoundError`"*
- ❌ *"Table-driven tests cover edges"* / *"Black-box tests with fake X"*

### 8. Epics only blocked by epics
Bd hard constraint. Task→epic block = wrong model → put block on specific child tasks (Rule 3).

### 9. Serial creates under same parent
Parallel `bd create --parent <same>` collides on hierarchical IDs, drops siblings. Same parent → sequential. Different parents (or no parent) → parallel.

### 10. Bd auto-closes molecules
Closing all children auto-closes epic. Don't add epic-blocked-by-children dep.

### 11. Non-blocking annotations
- `discovered-from` — only when literally discovered while working on linked issue.
- `related` — informational.
- `caused-by` — bug root cause.
- `supersedes` — replaces.
None affect `bd ready`.

### 12. Issue types
- `epic` — multi-task container.
- `feature` — user-visible functionality.
- `task` — internal work. **Default for children.**
- `bug` — broken. Use `caused-by` when root cause known.
- `chore` — tooling, deps, CI.

## havn description template

```markdown
<one-para summary + *why*>

## Scope
- <what this covers>

## Out of scope
- <only if ambiguous>

## Specs
- specs/<f>.md §<n>

## Wires up
- <what connects when this closes; domain-logic only>

## Acceptance
- <user-visible behavior or contract>
```

## Filing mechanics

```bash
bd create "<title>" \
  --type <epic|task|feature|bug|chore> \
  --priority <0-4> \
  --parent <id>            # if applicable
  --spec-id <spec-name>    # if maps to a spec file
  --description "<template>" \
  --json

bd dep add <dependent> <prereq>             # default: blocks
bd dep add <from> <to> --type discovered-from
bd dep add <from> <to> --type related

bd list
bd ready -n 100
```

## Anti-patterns (reject)

1. Phase ordering (Design → Impl → Test).
2. Separate "write tests" tasks.
3. `blocks` between siblings just to force a preferred execution chain or `bd ready` order.
4. Priority tiers to serialize within epic.
5. Epic-level `blocks` when only some children need prereq.
6. Skeleton/scaffolding tasks (create dir, add empty file).
7. TDD mechanics in descriptions.
8. `-f` markdown or `--graph` JSON batch flags — undocumented. Use `bd create` + `bd dep add`.

## Hard rules

- Never file before user confirms.
- Never claim work (`bd update --claim`) — impl agent's job.
- Never update existing issues via this skill — use `bd update` directly.
- Always `bd search` / `bd list` before filing to avoid dupes.
