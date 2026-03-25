---
name: needs-human
description: Block issue, ask specific question, move on. Use when you lack clarity or a decision — do not guess.
user-invocable: false
allowed-tools: Bash(bd *)
---

```bash
bd update <id> -s blocked --json
bd label add <id> human --json
bd comments add <id> "<specific question with options>" --json
```

Then `bd ready` and pick next task.

Comment must be a specific question, not "I'm confused."
Good: "Spec says X, code does Y. Follow spec or match existing?"
If no ready work remains, tell user what's blocked and why.
