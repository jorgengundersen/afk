# Walking Skeleton

Minimal structure to prove the CLI runs. One command, one output, no
internal packages yet.

## What the skeleton does

```
$ afk -p "hello world"
hello world
```

Parse `-p` flag, print it, exit 0. That's it.

## Structure

```
cmd/afk/main.go    # Entry point: parse -p flag, print prompt, exit
go.mod             # Module definition
```

## What main.go does

1. Define `-p` string flag.
2. Parse flags.
3. If `-p` is empty: print error to stderr, exit 2.
4. Print the prompt to stdout, exit 0.

No internal packages. No interfaces. No logging. Just a CLI that accepts
input and produces output.

## Out of scope

Everything else. Config structs, harnesses, loops, logging, beads, daemon
mode — all added incrementally on top of this skeleton.

## Definition of done

- `go build ./cmd/afk` produces a binary.
- `afk -p "hello world"` prints `hello world` and exits 0.
- `afk` with no `-p` prints an error to stderr and exits 2.
- `go test ./...` passes (main has a test).
