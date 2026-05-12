# Agent Instructions

## Active Context

The previous agent-specific implementation has been pruned. Treat the HTML
specs as the active source of product context:

- `specs/index.html`
- `specs/prd-simplified-afk.html`
- `specs/design-principles.html`

Do not infer requirements from deleted history, old releases, or obsolete
agent/prompt/harness workflows.

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
