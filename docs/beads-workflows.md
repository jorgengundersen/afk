# Beads Workflows

How this project uses bd for planning, bug reports, and human handoff.

## Planning: Spec Implementation / Feature / Refactor

Use an epic with child issues. Children have dependencies that enforce order.

**1. Create the epic:**

```bash
bd create "Implement harness abstraction" \
  -t epic -p 1 \
  -d "Add harness interface and Claude implementation per specs/afk-overview.md" \
  --spec-id specs/afk-overview.md \
  --json
```

**2. Break into ordered children:**

```bash
# First task — no deps
bd create "Define Harness interface and contract" \
  -t task -p 1 \
  --parent bd-<epic> \
  -d "Harness interface: Run(ctx, prompt) (int, error)" \
  --json

# Second task — depends on the first
bd create "Implement Claude harness" \
  -t task -p 1 \
  --parent bd-<epic> \
  -d "Concrete Claude implementation behind Harness interface" \
  --deps bd-<first> \
  --json
```

**3. Check plan readiness:**

```bash
bd ready --parent bd-<epic>    # shows unblocked children
bd epic status bd-<epic>       # shows completion progress
bd dep tree bd-<epic>          # visualize dependency graph
```

**Rules:**
- Keep children small (one commit each).
- Use `--deps` to encode ordering — `bd ready` surfaces what's unblocked.
- Link to spec with `--spec-id` so agents have context.
- Close the epic when `bd epic close-eligible` says it's ready.

## Bug Reports from Agents

Agents file bugs when they find something broken during other work.

```bash
bd create "Config validation accepts negative max-iterations" \
  -t bug -p 1 \
  -d "Validate() does not reject Config{MaxIterations: -1}. Expected exit 2." \
  --deps discovered-from:bd-<current-task> \
  --json
```

**Rules:**
- Always use `-t bug`.
- Always link with `discovered-from:<task-being-worked>` so there's a trail.
- Description must include: what's wrong, what was expected, how it was found.
- Don't fix the bug inline — file it, finish current work, let `bd ready` surface it.

## When an Agent Needs Human Input

If an agent hits ambiguity, missing context, or incorrect assumptions, it
should not guess. Instead: label the issue as blocked, add a comment
explaining what's needed, and move on.

**1. Block the issue and explain:**

```bash
bd update bd-<id> -s blocked --json
bd label add bd-<id> needs-human
bd comments add bd-<id> "Spec says harness returns exit code, but Claude \
  stream-json output includes tool results. Should we parse those or just \
  use the process exit code? Need human decision."
```

**2. Move on to other work:**

```bash
bd ready   # find next unblocked task
```

**3. Human reviews and unblocks:**

```bash
bd human list                              # find all human-needed issues
bd show bd-<id>                            # read the question
bd human respond bd-<id> -r "<answer>"     # answers + closes in one step
```

**Rules:**
- Use status `blocked` + label `human` (not `needs-human` — must match `bd human list`).
- The comment must be a specific question, not "I'm confused." State what
  decision is needed and what the options are.
- Never stall a session waiting for human input. File it, move on.
