# Instructions

## Development Practice

**Red/green TDD** — one failing test, one implementation, commit. No code without a failing test first. TDD applies to code, not documentation.

## Autonomy

You own implementation decisions: function signatures, module placement, naming,
internal decomposition. Use specs and `specs/design-principles.md` as guidelines,
not dictation.

- Read the task's linked spec for context on what you're building
- The task description is your scope — don't expand beyond it
- If a task is larger than expected, file a new issue for the excess
- Make decisions and move — don't ask for permission on internals

## Workflow

- run `bd ready --json`
- pick the most important issue and claim it
- execute task with red/green TDD. One test then one implementation at a time.
- make sure to commit the work you have done to trigger pre-commit hook
- fix any issues
- file bugs you encounter: file-bug skill
- if the issue is unclear or needs human review: needs-human skill

## Landing the Plane

Work is NOT complete until `git push` succeeds.

1. **File issues for remaining work** — create issues for anything that needs follow-up
2. **Run quality gates** — tests, linters, builds must pass
3. **Update issue status** — close finished work, update in-progress items
4. **Push to remote:**
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Verify** — all changes committed AND pushed
6. **Exit** — provide context for next session

**Rules:**
- NEVER stop before pushing — that leaves work stranded locally
- NEVER say "ready to push when you are" — YOU must push
- If push fails, resolve and retry until it succeeds

