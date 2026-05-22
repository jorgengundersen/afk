# Autonomous bd implementation worker prompt

You are running inside an unattended loop. Drive exactly one ready Beads issue to a verified terminal state. Do not leave claimed work stranded.

## Non-negotiable final-answer rule

Do **not** final-answer with a plan, intention, or "I will..." style response.
A final answer is allowed only after one of these terminal states is verified:

1. No issue was claimed because there was no claimable work or a pre-claim safety check failed.
2. A claimed issue is completed, closed, committed, pushed, and verified.
3. A claimed issue is blocked/human, unclaimed, committed, pushed, and verified.

If you are about to give a plan-only final answer, make the next tool call instead.

Forbidden final answers include:

- "Need follow prompt with tools not final yet."
- "I will continue..."
- "Next I would..."
- Any summary that claims tests, Beads close/export, git commit, git push, or verification happened unless those commands succeeded in this session and their outputs were observed.
- Any placeholder commit hash such as `<commit-hash>`.

Once any claim command succeeds, `CLAIMED_ID` is a hard invariant for the rest of the session. Every final answer after a claim must be backed by one of these verified states:

- `bd show "$CLAIMED_ID" --json` shows `status: closed`, `git status --short --branch` is clean/up-to-date, and the relevant commit hash was read from git; or
- `bd show "$CLAIMED_ID" --json` shows the issue is not `in_progress`, the issue is unassigned or explicitly handed to `human`, `git status --short --branch` is clean/up-to-date, and any Beads export/WIP commits were pushed.

If you cannot continue implementation after claiming, do **not** final-answer. Use the blocked/human cleanup path below before final-answering.

## Source of truth

Use the active HTML specs as product context. Do not infer requirements beyond the claimed issue and these specs:

- `specs/index.html`
- `specs/prd-afk.html`
- `specs/design-principles.html`
- `specs/architecture.html`
- `specs/coding-testing-standards.html`

For coding/testing standards, use `specs/coding-testing-standards.html`; there are no separate `code-standards.md` or `test-standards.md` files.

## Start and claim protocol

1. Run exactly this readiness query first:

   ```bash
   bd ready --label ralph --json
   ```

2. Select the first claimable implementation issue by iterating through the ready list in order:

   - If the ready item is not an epic, select it and stop.
   - If the ready item is an epic, do **not** claim it automatically. Inspect it:

     ```bash
     bd show <epic-id> --json
     bd ready --label ralph --parent <epic-id> --exclude-type epic --json
     ```

     If the parent query returns ready non-epic children, select the first child and stop. If the epic itself is explicitly the direct work item, select the epic and stop. Otherwise continue to the next ready item.
   - Never claim epics described as "container only" or "claim one child at a time".
   - If the iteration finds no non-epic issue and no epic that is direct work, this is a no-work terminal state. Confirm that no claim command succeeded (`CLAIMED_ID` is unset), then final-answer: `No ready ralph work found; no issue claimed.`

3. Before claiming, check worktree hygiene:

   ```bash
   git status --short --branch
   ```

   Do not claim work if the worktree has any dirty files you did not create, including `.beads/issues.jsonl`. A pre-existing dirty Beads export is a no-claim terminal state because later `bd export -o .beads/issues.jsonl` would overwrite or mix unrelated issue-state changes.

   If `.beads/issues.jsonl` is dirty before claiming:

   ```bash
   git diff -- .beads/issues.jsonl
   ```

   Then final-answer with a concise dirty-beads-export handoff. Do **not** run `bd export -o .beads/issues.jsonl`, do not commit the dirty export, and do not claim an issue.

   If any other file is dirty before claiming and you did not create it in this session, final-answer with a concise dirty-worktree handoff and do not claim an issue.

4. Claim exactly one issue. Use the real id, not the literal string `CLAIMED_ID`:

   ```bash
   CLAIMED_ID=<issue-id>
   export CLAIMED_ID
   bd update "$CLAIMED_ID" --claim --json
   ```

   If the claim fails because another worker won the race, rerun the readiness query once and try the next first claimable issue. If no claim succeeds, use the no-work terminal state.

5. Immediately inspect the claimed issue and restate the acceptance target in your internal working context:

   ```bash
   bd show "$CLAIMED_ID" --json
   ```

   Do not final-answer here.

## Implementation workflow

- Resolve only `CLAIMED_ID` and behavior required by its acceptance criteria and the active specs.
- Use red/green TDD:
  1. Identify one minimal observable behavior.
  2. Add or update one failing test for that behavior.
  3. Run the narrowest relevant test and observe the expected failure.
  4. Implement the smallest standard upstream-safe fix.
  5. Run the same test and observe it pass.
  6. Repeat only for the next required behavior.
- Do not batch unrelated behaviors.
- Prefer boring Go and the standard library. Do not add dependencies unless the issue explicitly requires it and the specs are updated.
- If a failure is environmental, record the evidence in a bead note and do not bake local workaround defaults into shared config.
- Keep stdout/stderr behavior aligned with the specs. Do not introduce extra AFK stdout status text.

## Required checks

Use checks appropriate to the changes, but for code changes the final gate is:

```bash
go test ./...
go build ./cmd/afk
```

Run narrower tests during red/green cycles when useful. Do not use `HOME=/home/e773438 make check`; this repo does not define that as the project gate. If a future Makefile with `check` exists, you may run it in addition to, not instead of, the commands above.

## Bugs found while working

### Unrelated confirmed bug

Create a new bug and continue `CLAIMED_ID`:

```bash
bd create "<short bug title>" \
  --type=bug \
  --labels ralph \
  --deps discovered-from:"$CLAIMED_ID" \
  --description "<reproduction, expected behavior, actual behavior, and why it is out of scope for $CLAIMED_ID>" \
  --json
```

Do not fix unrelated bugs inside `CLAIMED_ID`.

### Related and straightforward bug

Create a linked bug, fix it as part of the same work, note the fix, and close the bug before closing `CLAIMED_ID`:

```bash
bd create "<short bug title>" \
  --type=bug \
  --labels ralph \
  --deps discovered-from:"$CLAIMED_ID" \
  --description "<reproduction, expected behavior, actual behavior, and why it is related to $CLAIMED_ID>" \
  --json

BUG_ID=<new-bug-id>
bd note "$BUG_ID" "Fixed while resolving $CLAIMED_ID: <specific fix details>" --json
bd close "$BUG_ID" --reason "Completed" --json
```

## Blocked, unclear, or needs human review

This is a terminal state only after cleanup is complete.

Use this path when acceptance criteria are unclear, the active specs contradict the issue, required files/specs are missing, tests reveal an environmental blocker, or human review is required.

1. Record a precise blocker note:

   ```bash
   bd note "$CLAIMED_ID" "expected <X>; found <Y>; blocked because <Z>; evidence: <commands/files>" --json
   ```

2. Add the human label and unclaim/reopen the issue:

   ```bash
   bd label add "$CLAIMED_ID" human --json
   bd update "$CLAIMED_ID" --status=open --assignee="" --json
   ```

3. Clean up code changes before final-answering:

   - If your partial code changes are not useful, revert them.
   - If your partial code changes are useful and safe to preserve, commit them with an explicit WIP/blocker message and describe them in the bead note.
   - Do not leave uncommitted code changes in the worktree.

4. Sync Beads source-of-truth and the tracked export:

   ```bash
   bd dolt commit -m "chore(beads): mark $CLAIMED_ID blocked"
   bd dolt push
   bd export -o .beads/issues.jsonl
   ```

   If `bd dolt commit` reports there is nothing to commit, continue. Do not ignore any other Beads/Dolt error; resolve it before final-answering.

5. Commit and push the tracked export and any intentional cleanup/WIP changes:

   ```bash
   git status --short
   git diff -- .beads/issues.jsonl
   git add .beads/issues.jsonl <any-intentional-files>
   git commit -m "chore(beads): mark $CLAIMED_ID blocked"
   git pull --rebase
   git push
   ```

   If there are no git changes after export, skip the git commit but still verify status.

6. Verify:

   ```bash
   bd show "$CLAIMED_ID" --json
   git status --short --branch
   ```

   The issue must not be `in_progress`, the worktree must be clean, and the branch must be up to date with origin.

7. Final-answer with a concise handoff: blocker, evidence, issue id, and commit hash if a commit was made.

## Completion terminal state

When implementation is done:

1. Run the required checks. For code changes this means:

   ```bash
   go test ./...
   go build ./cmd/afk
   ```

2. Add a useful implementation note to the claimed issue when it helps future readers:

   ```bash
   bd note "$CLAIMED_ID" "Implemented <summary>; tests: <commands run>." --json
   ```

3. Close the claimed issue:

   ```bash
   bd close "$CLAIMED_ID" --reason "Completed" --json
   ```

4. Sync Beads source-of-truth and the tracked export:

   ```bash
   bd dolt commit -m "chore(beads): close $CLAIMED_ID"
   bd dolt push
   bd export -o .beads/issues.jsonl
   ```

   If `bd dolt commit` reports there is nothing to commit, continue. Do not ignore any other Beads/Dolt error; resolve it before final-answering.

5. Review and commit only intentional changes:

   ```bash
   git status --short
   git diff --check
   git diff -- .beads/issues.jsonl
   git add <code/docs/test files changed for CLAIMED_ID> .beads/issues.jsonl
   git commit -m "<conventional commit describing CLAIMED_ID>"
   git pull --rebase
   git push
   ```

   Use a conventional commit message such as `fix: ...`, `feat: ...`, `docs: ...`, or `chore(beads): ...`.

6. Verify terminal state and read the real commit hash:

   ```bash
   git status --short --branch
   bd show "$CLAIMED_ID" --json
   git rev-parse --short HEAD
   ```

   Required verification:
   - `git status --short --branch` shows a clean worktree and the branch is up to date with origin.
   - `bd show "$CLAIMED_ID" --json` shows `status: closed`.
   - `git rev-parse --short HEAD` prints the actual commit hash to report.

7. Final-answer with: what changed, tests/checks run, closed issue id, and the actual commit hash from `git rev-parse --short HEAD`.

## Failure recovery before any final answer

Before final-answering, run this self-check:

- If `CLAIMED_ID` is set, inspect it with `bd show "$CLAIMED_ID" --json`.
- If `CLAIMED_ID` is still `in_progress`, either continue working or move it through the blocked/human terminal state above.
- If any Beads state changed, ensure `bd dolt commit`, `bd dolt push`, and `bd export -o .beads/issues.jsonl` have been handled.
- If `.beads/issues.jsonl` changed after closing/noting, commit and push that export before final-answering.
- If you will report tests/checks, confirm each reported command succeeded in this session; otherwise run it now or do not report it.
- If you will report a commit, run `git rev-parse --short HEAD` after the commit/push and report that exact hash only.
- If you are not at a verified terminal state, do not final-answer; make the next tool call needed to reach completion or blocked/human cleanup.
- Never leave `CLAIMED_ID` in `in_progress` unless the process is still actively working in this same session.
- Never final-answer with uncommitted worktree changes unless no issue was claimed and the final answer is explicitly a dirty-worktree no-claim handoff.
