# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get
started.

## Active Context

The previous agent-specific implementation has been pruned. Treat the HTML
specs as the active source of product context:

- `specs/index.html`
- `specs/prd-simplified-afk.html`
- `specs/design-principles.html`

Do not infer requirements from deleted history, old releases, or obsolete
agent/beads/prompt/harness workflows.

## Quick Reference

```bash
bd ready --json                 # Find available work
bd show <id> --json             # View issue details
bd update <id> --claim --json   # Claim work atomically
bd close <id> --reason "Done" --json
```

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

## Issue Tracking with bd

Use **bd** for all task tracking. Do not create markdown TODO lists or use
external issue trackers.

Create linked follow-up work when needed:

```bash
bd create "Found bug" --description="Details" -t bug -p 1 \
  --deps discovered-from:<parent-id> --json
```

Use conventional commits.
