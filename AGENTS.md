# Agent Instructions

## Active Context

Treat the HTML specs as the source of product context:

- `specs/index.html`
- `specs/prd-afk.html`
- `specs/design-principles.html`
- `specs/architecture.html`
- `specs/coding-testing-standards.html`

Do not infer requirements beyond the active specs.

## Non-Interactive Shell Commands

Always use non-interactive flags with file operations to avoid hanging on
confirmation prompts.

```bash
cp -f source dest
mv -f source dest
rm -f file
rm -rf directory
cp -rf source dest
```

Other commands that may prompt:

- `scp` - use `-o BatchMode=yes`
- `ssh` - use `-o BatchMode=yes`
- `apt-get` - use `-y`
- `brew` - use `HOMEBREW_NO_AUTO_UPDATE=1`

Use conventional commits.

## Beads Export Policy

Treat `.beads/issues.jsonl` as a git-trackable export for planning/history, not the Beads source of truth.

When planning changes or code changes materially update Beads issues, intentionally include `.beads/issues.jsonl` in the same commit.

Do not rely on automatic staging for this file. Review the diff and `git add .beads/issues.jsonl` explicitly when its issue-state changes are part of the commit.
