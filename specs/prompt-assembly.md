# Prompt Assembly

Pure function that composes the final prompt from a user prompt, an optional
beads issue, and an instruction string. No I/O — takes data in, returns a
string.

## What this adds

```
$ afk -p "refactor auth module"
# prompt = "refactor auth module"

$ afk --beads
# prompt = issue JSON + instruction to claim/complete/close

$ afk --beads -p "focus on tests"
# prompt = issue JSON + instruction + "focus on tests"

$ afk
# error: no prompt (caught by config validation, but assembly guards too)
```

## Structure

```
internal/prompt/assemble.go       # Assemble function
internal/prompt/assemble_test.go  # Tests
```

## Contract

Package `prompt` exposes a single assembly function. It accepts a user prompt
string (possibly empty), an optional issue JSON string (possibly empty), and
produces the final prompt sent to the harness.

When an issue is present, a hardcoded instruction is included that tells the
agent to claim the issue before starting work and close it when done. The
instruction references the issue ID from the JSON so the agent knows which
issue to operate on.

Assembly remains a pure function with no I/O.

### Behaviour rules

| Given | Then |
|-------|------|
| Prompt set, no issue | Prompt returned verbatim |
| Issue set, no prompt | Issue JSON + instruction returned |
| Both prompt and issue set | Issue JSON + instruction + user prompt returned |
| Neither prompt nor issue | Error: "no prompt provided" |
| Issue JSON is the full object | Injected as-is, not summarised or transformed |
| Whitespace-only prompt, no issue | Error (whitespace-only is not a valid prompt when no issue present) |
| Whitespace-only prompt with issue | Issue JSON + instruction returned (empty user prompt ignored) |

### What this does NOT do

- Read from stdin or any I/O source — callers provide strings.
- Fetch issues — the beads client does this.
- Template or customise the instruction string — it is hardcoded.
- Token counting or prompt truncation.

## Out of scope

- Stdin pipe as prompt source (caller reads stdin and passes it as prompt).
- Configurable instruction templates.
- Multi-issue composition (one issue per assembly call).

## Definition of done

- Assembly composes issue JSON + instruction + user prompt correctly for all
  input combinations.
- Assembly is a pure function with no I/O.
- `go test ./internal/prompt/...` passes.
