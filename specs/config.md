# Config & Validation

Parse CLI flags into a plain data struct. Validate constraints between flags.
No behaviour beyond parsing and validation — this is the foundation that the
loop, harnesses, and future config-file loading all build on.

## What this adds

```
$ afk -p "do the thing" -n 5 --harness claude
# Config struct populated, validated, passed to Run()

$ afk -p "do the thing" --harness codex
# Config struct populated, validated, passed to Run()

$ afk --raw "my-agent {prompt}" --harness claude
error: --raw cannot be combined with --harness or --model

$ afk --sleep 30s
error: --sleep requires daemon mode (-d)

$ afk
error: no prompt provided and beads not active; nothing to do

$ afk --beads --label backend --label-any bugfix --label-any feature
# Labels=["backend"], LabelsAny=["bugfix","feature"]

$ afk --label backend
error: --label requires --beads
```

Parse flags, validate constraints, exit 2 on validation failure. That's it.

## Structure

```
internal/config/config.go       # Config struct + ParseFlags + Validate
internal/config/config_test.go  # Tests for parsing and validation
cmd/afk/main.go                 # Thin entry point: ParseFlags → Validate → exit
```

## Config struct

Plain data. No methods that do work. No interfaces. No pointers to optional
fields — use zero values as "not set."

```go
type Config struct {
    Prompt      string
    MaxIter     int
    Daemon      bool
    Sleep       time.Duration
    Harness     string
    Model       string
    Raw         string
    HarnessArgs string
    Beads       bool
    Labels      []string
    LabelsAny   []string
}
```

## ParseFlags

`func ParseFlags(args []string) (Config, error)`

- Accepts `os.Args[1:]` (or test-provided args).
- Uses `flag.FlagSet` (not the global `flag` package) so it's testable and
  doesn't pollute global state.
- Returns a `Config` with values populated from flags + defaults.
- Defaults: `MaxIter=20`, `Sleep=60s`, `Harness="claude"`. Everything else
  is zero value.
- `--label` and `--label-any` are repeatable string flags that populate
  `Labels` and `LabelsAny` slices respectively.
- On parse error (unknown flag, bad type): return error.

## Validate

`func Validate(cfg Config) error`

Pure function. Takes a Config, returns nil or an actionable error message.

### Rules

| Condition                          | Error message                                                  |
|------------------------------------|----------------------------------------------------------------|
| `Raw` set + `Harness != "claude"`  | `--raw cannot be combined with --harness or --model`           |
| `Raw` set + `Model` set           | `--raw cannot be combined with --harness or --model`           |
| `Sleep` changed + `Daemon` false  | `--sleep requires daemon mode (-d)`                            |
| `Prompt` empty + `Beads` false    | `no prompt provided and beads not active; nothing to do`       |
| `Labels` set + `Beads` false      | `--label requires --beads`                                     |
| `LabelsAny` set + `Beads` false   | `--label-any requires --beads`                                 |

Note on the `--raw` + `--harness` check: since `Harness` defaults to
`"claude"`, we need a way to distinguish "user explicitly passed `--harness`"
from "default value." Track this with a `sleepSet` / `harnessSet` bool
alongside the Config, or use a separate `FlagSet.Visit` check inside
`ParseFlags` to record which flags were explicitly provided.

### What Validate does NOT check

- Whether the harness binary exists in PATH (that's a runtime check for the
  loop, not a config concern).
- Whether the harness name is known (same — belongs to the harness layer).

## What main.go becomes

```go
func main() {
    cfg, err := config.ParseFlags(os.Args[1:])
    if err != nil {
        fmt.Fprintln(os.Stderr, "error:", err)
        os.Exit(2)
    }
    if err := config.Validate(cfg); err != nil {
        fmt.Fprintln(os.Stderr, "error:", err)
        os.Exit(2)
    }
    // Future: pass cfg to Run()
    fmt.Println(cfg.Prompt)
}
```

Thin. Parse, validate, hand off. The skeleton's existing `-p` behaviour is
preserved — `afk -p "hello"` still prints "hello" and exits 0.

## Test cases

### ParseFlags

| Test                          | Input args                        | Expected                                    |
|-------------------------------|-----------------------------------|---------------------------------------------|
| defaults                      | `-p "hi"`                         | MaxIter=20, Sleep=60s, Harness="claude"     |
| all flags                     | `-p "x" -n 5 -d --sleep 30s ...` | All fields populated                        |
| unknown flag                  | `--nope`                          | Error                                       |
| bad type                      | `-n abc`                          | Error                                       |

### Validate

| Test                          | Config                                       | Expected                                             |
|-------------------------------|----------------------------------------------|------------------------------------------------------|
| valid with prompt             | `{Prompt: "x"}`                              | nil                                                  |
| valid with beads              | `{Beads: true}`                              | nil                                                  |
| raw + harness                 | `{Raw: "cmd", harnessSet: true}`             | error: --raw cannot be combined...                   |
| raw + model                   | `{Raw: "cmd", Model: "x"}`                  | error: --raw cannot be combined...                   |
| sleep without daemon          | `{Prompt: "x", sleepSet: true}`              | error: --sleep requires daemon mode                  |
| no prompt no beads            | `{}`                                         | error: no prompt provided...                         |

### Integration (main behaviour)

| Test                          | Args              | Stdout        | Stderr             | Exit |
|-------------------------------|-------------------|---------------|--------------------|------|
| prints prompt                 | `-p "hello"`      | `hello`       | —                  | 0    |
| no prompt no beads            | (none)            | —             | error: no prompt…  | 2    |

## Out of scope

- Stdin pipe as prompt source (future increment).
- Config file loading and merge (future — but this struct is designed for it).
- Harness registry or binary-exists check.
- The iteration loop or Run() function.
- Prompt assembly.

## Definition of done

- `go build ./cmd/afk` succeeds.
- `afk -p "hello"` prints `hello` and exits 0 (skeleton behaviour preserved).
- `afk` with no flags prints the "no prompt" error to stderr and exits 2.
- All validation rules produce correct error messages.
- `go test ./...` passes.
- No global `flag` state — `ParseFlags` uses a local `FlagSet`.
