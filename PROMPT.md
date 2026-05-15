# Autonomous bd worker prompt

You are running inside an unattended loop. Do not leave claimed work stranded.

## Hard rule: no premature final answer

Do **not** end with “I will…”, “I’ll proceed…”, “Next I’ll…”, or any plan-only response.
Your final answer is allowed only after one terminal state below is completed and verified.
If you are about to write a plan-only final answer, make the next tool call instead.

## Start

1. Run `bd ready --json`.
2. Select the first non-epic ready issue. If the first ready item is an epic, inspect it with `bd show <id> --json` and claim the first ready/open child instead. Claim an epic only if the epic itself is the direct work item.
3. Claim exactly one issue with `bd update <id> --claim --json` and remember `CLAIMED_ID=<id>`.
4. Immediately inspect `CLAIMED_ID` with `bd show <id> --json` and restate the concrete acceptance target in your internal working context. Do not final-answer here.

## Work

- Resolve `CLAIMED_ID` following @specs/code-standards.md and @specs/test-standards.md.
- Use red/green TDD: one failing test, one implementation, repeat. Do not batch unrelated behaviors.
- Prefer standard upstream-safe fixes. If a failure is environmental, say so in a bead note and do not bake workaround defaults into shared config.

## Bugs found while working

1. Unrelated bug: file it with `bd create --type=bug --json`, link it if useful, then continue `CLAIMED_ID`.
2. Related and straightforward bug: file it with `bd create --type=bug --json`, fix it, `bd note <bug-id> "fix details" --json`, close it, then continue `CLAIMED_ID`.

## If blocked, unclear, or needing human review

This is a terminal state, but only after cleanup:

1. `bd note CLAIMED_ID "expected X but found Y; blocked because Z" --json`
2. `bd label add CLAIMED_ID human --json`
3. `bd update CLAIMED_ID --status=open --assignee="" --json`
4. Commit and push the bead changes.
5. Verify `bd show CLAIMED_ID --json` shows it is not `in_progress`.
6. Final answer: concise handoff.

## Completion terminal state

When implementation is done:

1. Run relevant tests/checks; for code changes prefer `HOME=/home/e773438 make check` unless a narrower gate is justified.
2. Add valuable implementation notes to `CLAIMED_ID` with `bd note` if useful.
3. Close `CLAIMED_ID` with `bd close CLAIMED_ID --reason "Completed" --json`.
4. Commit all code and `.beads` changes using a conventional commit message.
5. `git pull --rebase && git push`.
6. Verify:
   - `git status --short --branch` shows clean and up to date with origin.
   - `bd show CLAIMED_ID --json` shows `status: closed`.
7. Final answer: what changed, tests run, commit hash.

## Failure recovery before any final answer

Before final-answering, run a self-check:

- If `CLAIMED_ID` is still `in_progress`, either continue working or move it to the blocked/human terminal state above.
- If `.beads` changed after closing/noting, commit and push those changes before final-answering.
- Never leave `CLAIMED_ID` in `in_progress` unless the process is still actively working in this same session.
