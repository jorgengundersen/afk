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
