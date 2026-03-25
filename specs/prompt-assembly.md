# Prompt Assembly

Pure function that takes a user prompt string and returns it. Foundation for
future prompt composition (beads issues, stdin pipe) — but right now, it's
just pass-through with validation.

## What this adds

```
$ afk -p "refactor auth module"
# prompt = "refactor auth module"

$ afk
# error: no prompt (caught by config validation, but assembly guards too)
```

## Structure

```
internal/prompt/assemble.go       # Assemble function
internal/prompt/assemble_test.go  # Tests
```

## Assemble

`func Assemble(prompt string) (string, error)`

- `prompt` is the user-provided prompt (may be empty).
- Returns the final prompt string or an error.

### Rules

| prompt | result |
|--------|--------|
| set    | prompt verbatim |
| empty  | error: "no prompt provided" |

The "empty" case should be caught by config validation before reaching
assembly. The error exists as a defensive guard — assembly does not assume
callers validated.

## Test cases

| Test                  | prompt              | expected output                |
|-----------------------|---------------------|--------------------------------|
| prompt provided       | "do stuff"          | "do stuff"                     |
| empty prompt          | ""                  | error: "no prompt provided"    |
| whitespace only       | "   "               | "   " (passed through as-is)   |

## Out of scope

- Beads issue context and prompt composition (future beads integration spec).
- Stdin pipe as prompt source (future — the caller reads stdin and passes it
  as `prompt`).
- Template languages or prompt customisation.
- Token counting or prompt truncation.

## Definition of done

- `Assemble` is a pure function with no I/O.
- All test cases pass.
- `go test ./internal/prompt/...` passes.
